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
// The state, the useCases of notes and vaults and the installation are the window's own, and each
// service is handed the ones it uses.
type windowWiring struct {
	cfg       Config
	a         pluginsWiring
	ports     registry.Ports
	db        *Index
	embedder  embeddingport.Embedder
	asking    embeddingport.Embedder
	inst      *editor.Installation
	state     *editor.State
	kinds     tabKinds
	server    *editor.FileServer
	switching windowsession.UserSwitch
	cases     WindowUseCases
}

// tabKinds are the use cases that name the kind of a file for the tabs of a window.
type tabKinds struct {
	files   *file.GetFileKinds
	chooser file.TabKindChooser
	list    *file.ListTabKinds
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

// WindowOpening is what OpenWindowWith opens a window over: the cache already open, the vault to show and
// where the window writes. Frame is the application window around the editor; none leaves the editor with no
// window of its own to title or to open for a tab. UseCases, when given, receives the use cases the window works
// through.
type WindowOpening struct {
	DB    *Index
	Named string
	Out   io.Writer

	Frame editor.Frame

	UseCases *WindowUseCases
}

// WindowUseCases are the use cases of notes and vaults a window works through. The notes the plugins work through, PluginNotes, pass no gate
// and report no write to the window.
type WindowUseCases struct {
	Notes       *NoteUseCases
	PluginNotes *NoteUseCases
	Queries     *query.Scenarios
	Listing     *registry.Listing
	Vaults      *VaultUseCases
	ShowTab     windowsession.ShowTab
}

// OpenWindowWith is OpenWindow over a cache already open, for a caller that opened it itself. The window
// closes the cache.
func OpenWindowWith(c Config, ctx context.Context, opening WindowOpening) (*OpenedWindow, error) {
	ports, err := c.registry()
	if err != nil {
		return nil, errors.Join(err, opening.DB.Close())
	}
	opened, err := openWindowWith(c, ctx, ports, opening, nil)
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

// windowBuilt is a window put together and not started yet, with the close of the embedder it opened.
type windowBuilt struct {
	wiring        *windowWiring
	closeEmbedder func() error
}

func openWindowWith(
	c Config, ctx context.Context, ports registry.Ports, opening WindowOpening, texts *TextVersions,
) (*windowWiring, error) {
	first, err := registry.NewGetVaultToShow(ports.Lookup, ports.Vaults, ports.Last).Execute(opening.Named)
	if err != nil {
		return nil, errors.Join(err, opening.DB.Close())
	}
	window := buildWindow(ctx, c, ports, opening, texts)
	shareUseCases(opening, window.wiring.cases)
	if err := startWindow(ctx, window.wiring, first); err != nil {
		return nil, errors.Join(err, window.closeEmbedder(), opening.DB.Close())
	}
	return window.wiring, nil
}

func shareUseCases(opening WindowOpening, cases WindowUseCases) {
	if opening.UseCases != nil {
		*opening.UseCases = cases
	}
}

func buildWindow(
	ctx context.Context, c Config, ports registry.Ports, opening WindowOpening, texts *TextVersions,
) windowBuilt {
	a := assemble(c, texts)
	db := opening.DB

	// One task list serves every operation and the window that displays it.
	// A request is one task regardless of how it was issued. The list is created
	// before any component that reports into it.
	tasks := run.NewTasks()
	reportPluginFailures(a, tasks)
	reportDependencies(a, tasks)
	setPluginWork(a, tasks, &run.Lock{})

	// Opened once, for as long as the window is. A local model is fetched and
	// compiled behind this, so the window is drawn while it arrives.
	embedder, asking, closeEmbedder, why := searchEmbedders(a.search, ctx, tasks)
	embedding := indexing.NewEmbedding(embedder, embedder, db.ChunkVectors())
	state := editor.NewState(tasks, opening.Frame, func(event string) { emitPluginEvent(a, event) })
	state.SetIndexing(embedding)
	indexer := newIndexWorker(a, db, state, embedding, opening.Out)

	// A note saved by the user is refreshed in the index before the save returns,
	// so the vault reflects its content without waiting for the watcher.
	current := &vault.CurrentRefresh{Fallback: func() vault.Refreshing { return GetRefresh(c, db) }}
	cases := newWindowUseCases(a, ports, db, state, current)

	session := newWindowSession(c, a, state, ctx, windowParts{
		db: db, indexer: indexer, embedding: embedding, moves: cases.Vaults, current: current, out: opening.Out,
		embedderError: why, closeEmbedder: closeEmbedder,
	})
	cases.ShowTab = session.tools.showTab
	inst := editor.NewInstallation(state, editor.Lifecycle{
		Start: session.cases.start, Show: session.cases.show, OpenPlugin: session.cases.openPlugin,
		Flush: session.cases.flush, Close: session.cases.close,
		NoteWritten: indexer.WakeNotes,
	}, session.tools.importFiles, opening.Out)

	kinds := newTabKinds(a, db)
	return windowBuilt{
		wiring: &windowWiring{
			cfg: c, a: a, ports: ports, db: db, embedder: embedder, asking: asking, inst: inst, state: state,
			kinds: kinds, server: newFileServer(a, state, cases), switching: session.userSwitch, cases: cases,
		},
		closeEmbedder: closeEmbedder,
	}
}

func newWindowUseCases(
	a pluginsWiring, ports registry.Ports, db *Index, state *editor.State, current *vault.CurrentRefresh,
) WindowUseCases {
	notes := vaultNotes(
		a.vault, db, vault.NewUpdateIndex(current.Get).Execute,
		noteViews{
			Moves:  note.NewTellMoveInShownVault(state.GetShownVault, windowView(state)),
			Writes: state.Writes,
			Follow: state.Writes,
		},
	)
	queries := vaultQueries(db)
	pluginNotes := vaultNotes(
		a.vault, db, vault.NewUpdateIndex(current.Get).Execute,
		noteViews{
			Moves:  note.NewTellMoveInShownVault(state.GetShownVault, windowView(state)),
			Writes: note.OpenGate{},
		},
	)
	moves := vaultVaults(a.vault, ports, db, notes.Place.Move)
	listing := vaultListing(a.vault, ports)
	return WindowUseCases{Notes: &notes, Queries: &queries, PluginNotes: &pluginNotes, Listing: &listing, Vaults: &moves}
}

func newTabKinds(a pluginsWiring, db *Index) tabKinds {
	files := file.NewGetFileKinds(vaultReaders(a.vault), db.NoteQueries())
	chooser := getTabKindChooser(a)
	return tabKinds{files: files, chooser: chooser, list: file.NewListTabKinds(files, chooser)}
}

func newFileServer(a pluginsWiring, state *editor.State, cases WindowUseCases) *editor.FileServer {
	return editor.NewFileServer(
		state,
		file.NewOpenVaultFile(
			&cases.Listing.Find, file.NewOpenFile(vaultReaders(a.vault)), openPluginFile(a.vault), getMediaType(a.vault),
		),
	)
}

func startWindow(ctx context.Context, w *windowWiring, first domain.Vault) error {
	mountServices(w)
	return w.inst.Start(ctx, first, w.server)
}

func mountServices(w *windowWiring) {
	mountVault(w)
	mountWorkspace(w)
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
}

func windowView(state *editor.State) windowsession.View {
	notices := state.Notices
	return windowsession.NewView(&notices.Places, &notices.Edits, &notices.ChangeEvents)
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
