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

// GetUnknownVaultName returns the name of the vault, named at start, that err says the registry does not
// hold.
func GetUnknownVaultName(err error) (string, bool) {
	unknown, ok := errors.AsType[*registry.UnknownError](err)
	if !ok {
		return "", false
	}
	return unknown.Name, true
}

// windowWiring is what the wire_window_<service>.go files share while one editor window is built.
// The state, the scenarios of notes and vaults and the installation are the window's own, and each
// service is handed the ones it uses.
type windowWiring struct {
	cfg   Config
	ports registry.Ports
	db    *Index

	embedder embeddingport.Embedder
	asking   embeddingport.Embedder
	inst     *editor.Installation
	tabKinds *file.ListTabKinds
	server   *editor.FileServer
	wrap     func(windowsession.VaultPassesFunc) windowsession.VaultPassesFunc
	cases    *WindowUseCases
}

// OpenWindow opens the registry and the cache, builds the services of the editor window over
// them, and opens the vault named, or the one shown last when none is named.
func OpenWindow(c Config, ctx context.Context, named string, out io.Writer) (*editor.Installation, error) {
	db, err := openIndex(vaultOf(c), ctx)
	if err != nil {
		return nil, err
	}
	return OpenWindowOver(c, ctx, db, named, out)
}

// OpenWindowOver is OpenWindow over a cache already open, for a caller that opened
// it itself. The window closes the cache.
func OpenWindowOver(c Config,
	ctx context.Context, db *Index, named string, out io.Writer,
) (*editor.Installation, error) {
	return OpenWindowWith(c, ctx, WindowOpening{DB: db, Named: named, Out: out})
}

// WindowOpening is what OpenWindowWith opens a window over: the cache already open,
// the vault to show, where the window writes, and an optional function the walk of the vault goes through.
type WindowOpening struct {
	DB    *Index
	Named string
	Out   io.Writer
	Wrap  func(windowsession.VaultPassesFunc) windowsession.VaultPassesFunc

	UseCases *WindowUseCases
}

// WindowUseCases are the use cases of notes and vaults a window works through.
type WindowUseCases struct {
	Notes   *note.Scenarios
	Queries *query.Scenarios
	Listing *registry.Listing
	Vaults  *vault.Scenarios
	ShowTab windowsession.ShowTab
}

// OpenWindowWith is OpenWindowOver with the walk of the vault put through Wrap, for a caller
// that holds one back or drives it. A nil Wrap leaves it as it is.
func OpenWindowWith(c Config, ctx context.Context, opening WindowOpening) (*editor.Installation, error) {
	ports, err := c.registry()
	if err != nil {
		return nil, errors.Join(err, opening.DB.Close())
	}
	opened, err := openWindowWith(c, ctx, ports, opening)
	if err != nil {
		return nil, err
	}
	return opened.inst, nil
}

func openWindowWith(c Config, ctx context.Context, ports registry.Ports, opening WindowOpening) (*windowWiring, error) {
	w := &windowWiring{cfg: c, ports: ports, db: opening.DB, wrap: opening.Wrap, cases: getWindowUseCases(opening)}
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
	embedding := indexing.NewEmbedding(embedder, w.db.ChunkVectors())
	state := editor.NewState(tasks)
	state.Events.Emit = func(event string) { emitPluginEvent(plugins(w.cfg), event) }
	state.SetIndexing(embedding)
	indexer := newIndexWorker(w, state, embedding, out)
	state.Writes.NotifyNoteWritten = indexer.WakeNotes

	// A note saved by the user is refreshed in the index before the save returns,
	// so the vault reflects its content without waiting for the watcher.
	current := &vault.CurrentRefresh{Fallback: func() vault.Refresh { return GetRefresh(w.cfg, w.db) }}
	notes := vaultNotes(
		vaultOf(w.cfg), w.db, vault.NewUpdateIndex(current.Get).Execute,
		noteViews{Moves: showMovesIn(state.GetShownVault, windowView(state))},
	)
	queries := vaultQueries(w.db)
	w.cases.Notes, w.cases.Queries = &notes, &queries
	moves := vaultVaults(vaultOf(w.cfg), w.ports, w.db, w.cases.Notes.Place.Move)
	w.cases.Vaults = &moves
	listing := vaultListing(vaultOf(w.cfg), w.ports)
	w.cases.Listing = &listing

	session := windowsession.NewScenarios(&state.Runtime, windowsession.Deps{
		Window: windowsession.WindowPorts{
			Pages: state.Window, Writes: state.Writes, Tasks: state.Window, Changes: &state.Notices.ChangeEvents, TabFront: &state.Notices.TabIDs,
		},
		Vault: windowsession.VaultPorts{
			Import: moves.Tree.Import, Identity: vault.NewCheckIdentity(vaultIdentity(vaultOf(w.cfg))), Recorded: moves.RecordOpen, Passes: wrapPasses(w, startVault(w.cfg, w.db, current)),
			Texts: getTextVersions(w.cfg),
		},
		Index: windowsession.IndexPorts{
			Wake: indexer, NewSession: newIndexSession(indexer), Triggers: getTriggerSource(w.cfg), Model: embedding,
		},
		Shutdown: windowsession.ShutdownPorts{
			Door: state.Gate,
			Closers: []windowsession.Closer{
				windowsession.CloseFunc(state.ClosePlaying),
				windowsession.CloseFunc(func(context.Context) error { return closeEmbedder() }),
				windowsession.CloseFunc(func(context.Context) error { return w.db.Close() }),
			},
		},
		Out:           out,
		EmbedderError: why,
		Closing:       ctx.Done(),
		IsRebuilding:  w.cfg.ShouldRebuildIndex,
	})
	inst := editor.NewInstallation(state, editor.Lifecycle{
		Start: session.Start, Show: session.Switching.Show, OpenPlugin: session.Switching.OpenPlugin,
		Flush: session.Closing.Flush, Close: session.Closing.Close,
	}, session.Import, out)
	w.inst = inst
	w.cases.ShowTab = session.ShowTab

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

// GetWindowView returns the window as the core asks it to show a place, a move, an edit or a tab.
func GetWindowView(inst *editor.Installation) windowsession.View {
	return windowView(inst.State)
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
