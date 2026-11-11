package container

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// windowWiring is what the wire_window_<service>.go files share while one editor window is built.
// The state, the scenarios of notes and vaults and the installation are the window's own, and each
// service is handed the ones it uses.
type windowWiring struct {
	cfg   Config
	ports registry.Ports
	db    *Index
	frame editor.Frame

	embedder embeddingport.Embedder
	asking   embeddingport.Embedder
	inst     *editor.Installation
	state    *editor.State
	tabKinds *file.ListTabKinds
	server   *editor.FileServer
	wrap     func(windowsession.VaultPassesFunc) windowsession.VaultPassesFunc
	cases    *WindowUseCases
}

// OpenWindow opens the registry and the cache, builds the services of the editor window over
// them, and opens the vault named, or the one shown last when none is named.
func OpenWindow(c Config, ctx context.Context, named string, out io.Writer) (*OpenedWindow, error) {
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return nil, err
	}
	return OpenWindowWith(c, ctx, WindowOpening{DB: db, Named: named, Out: out})
}

// WindowOpening is what OpenWindowWith opens a window over: the cache already open,
// the vault to show, where the window writes, and an optional function the walk of the vault goes through.
// Frame is the application window around the editor; none leaves the editor with no window of its own to
// title or to open for a tab.
type WindowOpening struct {
	DB    *Index
	Named string
	Out   io.Writer
	Wrap  func(windowsession.VaultPassesFunc) windowsession.VaultPassesFunc

	Frame editor.Frame

	UseCases *WindowUseCases
}

// WindowUseCases are the use cases of notes and vaults a window works through.
type WindowUseCases struct {
	Notes   *NoteScenarios
	Queries *query.Scenarios
	Listing *registry.Listing
	Vaults  *VaultScenarios
	ShowTab windowsession.ShowTab
}

// OpenWindowWith is OpenWindow over a cache already open, for a caller that opened it itself, with the walk of
// the vault put through Wrap, for a caller that holds one back or drives it. A nil Wrap leaves it as it is.
// The window closes the cache.
func OpenWindowWith(c Config, ctx context.Context, opening WindowOpening) (*OpenedWindow, error) {
	ports, err := c.registry()
	if err != nil {
		return nil, errors.Join(err, opening.DB.Close())
	}
	opened, err := openWindowWith(c, ctx, ports, opening)
	if err != nil {
		return nil, err
	}
	return getOpenedWindow(opened), nil
}

// OpenedWindow is an editor window opened: the installation that serves it and the state its services work
// through.
type OpenedWindow struct {
	*editor.Installation
	State *editor.State
}

func getOpenedWindow(w *windowWiring) *OpenedWindow {
	return &OpenedWindow{Installation: w.inst, State: w.state}
}

func openWindowWith(c Config, ctx context.Context, ports registry.Ports, opening WindowOpening) (*windowWiring, error) {
	w := &windowWiring{
		cfg: c, ports: ports, db: opening.DB, frame: opening.Frame, wrap: opening.Wrap, cases: getWindowUseCases(opening),
	}
	return openWindow(ctx, w, opening.Named, opening.Out)
}

func openWindow(ctx context.Context, w *windowWiring, named string, out io.Writer) (*windowWiring, error) {
	first, err := registry.NewGetVaultToShow(w.ports.Lookup, w.ports.Vaults, w.ports.Last).Execute(named)
	if err != nil {
		return nil, errors.Join(err, w.db.Close())
	}

	// One task list serves every operation and the window that displays it.
	// A request is one task regardless of how it was issued. The list is created
	// before any component that reports into it.
	tasks := run.NewTasks()
	models := &run.Lock{}
	reportPluginFailures(plugins(w.cfg), tasks)
	reportDependencies(plugins(w.cfg), tasks)
	setPluginWork(plugins(w.cfg), tasks, models)

	// Opened once, for as long as the window is. A local model is fetched and
	// compiled behind this, so the window is drawn while it arrives.
	embedder, asking, closeEmbedder, why := searchEmbedders(searchOf(w.cfg), ctx, tasks)
	w.embedder, w.asking = embedder, asking
	embedding := indexing.NewEmbedding(embedder, embedder, w.db.ChunkVectors())
	state := editor.NewState(tasks, w.frame, func(event string) { emitPluginEvent(plugins(w.cfg), event) })
	state.SetIndexing(embedding)
	indexer := newIndexWorker(w, state, embedding, out)

	// A note saved by the user is refreshed in the index before the save returns,
	// so the vault reflects its content without waiting for the watcher.
	current := &vault.CurrentRefresh{Fallback: func() vault.Refreshing { return GetRefresh(w.cfg, w.db) }}
	notes := vaultNotes(
		vaultOf(w.cfg), w.db, vault.NewUpdateIndex(current.Get).Execute,
		noteViews{Moves: note.NewTellMoveInShownVault(state.GetShownVault, windowView(state)), Writes: state.Writes},
	)
	queries := vaultQueries(w.db)
	w.cases.Notes, w.cases.Queries = &notes, &queries
	moves := vaultVaults(vaultOf(w.cfg), w.ports, w.db, w.cases.Notes.Place.Move)
	w.cases.Vaults = &moves
	listing := vaultListing(vaultOf(w.cfg), w.ports)
	w.cases.Listing = &listing

	session, tools := windowSession(w, state, ctx, windowParts{
		indexer: indexer, embedding: embedding, moves: &moves, current: current, out: out,
		embedderError: why, closeEmbedder: closeEmbedder,
	})
	w.inst = editor.NewInstallation(state, editor.Lifecycle{
		Start: session.start, Show: session.show, OpenPlugin: session.openPlugin,
		Flush: session.flush, Close: session.close,
		NoteWritten: indexer.WakeNotes,
	}, tools.importFiles, out)
	w.state = state
	w.cases.ShowTab = tools.showTab

	if err := startWindow(ctx, w, first); err != nil {
		return nil, errors.Join(err, closeEmbedder(), w.db.Close())
	}
	return w, nil
}

func startWindow(ctx context.Context, w *windowWiring, first domain.Vault) error {
	if err := mountServices(w); err != nil {
		return err
	}
	return w.inst.Start(ctx, first, w.server)
}

func wrapPasses(w *windowWiring, passes windowsession.VaultPassesFunc) windowsession.VaultPassesFunc {
	if w.wrap == nil {
		return passes
	}
	return w.wrap(passes)
}

func mountServices(w *windowWiring) error {
	mountVault(w)
	if err := mountWorkspace(w); err != nil {
		return err
	}
	mountWindowState(w)
	mountFile(w)
	mountNote(w)
	mountSearch(w)
	mountVaults(w)
	mountSettings(w)
	mountMedia(w)
	mountAgent(w)
	mountCommand(w)
	mountText(w)
	mountPlugin(w)
	mountCatalog(w)
	return nil
}

func windowView(state *editor.State) windowsession.View {
	notices := state.Notices
	return windowsession.NewView(&notices.Places, &notices.Edits, &notices.ChangeEvents)
}

func getWindowUseCases(opening WindowOpening) *WindowUseCases {
	if opening.UseCases != nil {
		return opening.UseCases
	}
	return &WindowUseCases{}
}

// GetUnknownVaultName returns the name of the vault, named at start, that err says the registry does not
// hold.
func GetUnknownVaultName(err error) (string, bool) {
	unknown, ok := errors.AsType[*registry.UnknownError](err)
	if !ok {
		return "", false
	}
	return unknown.Name, true
}
