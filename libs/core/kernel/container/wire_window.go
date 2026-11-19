package container

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexgeneration"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	logsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// windowWiring is what the wire_window_<service>.go files share while one editor window is built.
// The state, the useCases of notes and vaults and the installation are the window's own, and each
// service is handed the ones it uses.
type windowWiring struct {
	logEntries  *logsusecase.Entries
	a           pluginsWiring
	ports       registry.Ports
	db          *Index
	embedder    searchsourceport.Source
	asking      searchsourceport.Source
	inst        *editor.Installation
	state       *editor.State
	kinds       tabKinds
	server      *editor.FileServer
	switching   windowsession.UserSwitch
	cases       WindowUseCases
	tasks       *run.Tasks
	generations *indexgeneration.Generations
}

// tabKinds are the use cases that name the kind of a file for the tabs of a window.
type tabKinds struct {
	files   *file.GetFileKinds
	chooser file.TabKindChooser
	list    *file.ListTabKinds
}

// OpenWindow opens the registry and the cache, builds the services of the editor window over
// them, and opens the vault named, or the one shown last when none is named.
func (c Config) OpenWindow(exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, ctx context.Context, opening WindowOpening) (*OpenedWindow, error) {
	db, err := OpenIndex(c, res, ctx)
	if err != nil {
		return nil, err
	}
	return openWindowOver(plugins(c, exec, prefs, endpoints, res), res, ctx, db, opening)
}

func openWindowOver(a pluginsWiring, res *Resources, ctx context.Context, db *Index, opening WindowOpening) (*OpenedWindow, error) {
	ports, err := vaultRegistry(a.vault.RegistryPath)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	opened, err := openWindowWith(ctx, a, getLogEntries(res), ports, db, opening)
	if err != nil {
		return nil, err
	}
	return getOpenedWindow(opened), nil
}

// WindowOpening names the vault to show, the output and the optional editor frame.
type WindowOpening struct {
	Named string
	Out   io.Writer
	Frame editor.Frame
}

// WindowUseCases are the use cases of notes and vaults a window works through. The notes the plugins work through, PluginNotes, pass no gate
// and report no write to the window.
type WindowUseCases struct {
	Notes       *NoteUseCases
	PluginNotes *NoteUseCases
	Queries     *query.UseCases
	Listing     *registry.Listing
	Vaults      *VaultUseCases
	ShowTab     windowsession.ShowTab
	OpenPlugin  windowsession.OpenPluginVault
}

// OpenedWindow is an editor window opened: the installation that serves it, the state its services work
// through, the cache it reads and the use cases it works through.
type OpenedWindow struct {
	*editor.Installation
	State    *editor.State
	Index    *Index
	UseCases WindowUseCases
}

func getOpenedWindow(w *windowWiring) *OpenedWindow {
	return &OpenedWindow{Installation: w.inst, State: w.state, Index: w.db, UseCases: w.cases}
}

// windowBuilt is a window put together and not started yet, with the close of the embedder it opened.
type windowBuilt struct {
	wiring        *windowWiring
	closeEmbedder func() error
}

func openWindowWith(
	ctx context.Context, a pluginsWiring, entries *logsusecase.Entries, ports registry.Ports, db *Index, opening WindowOpening,
) (*windowWiring, error) {
	first, err := registry.NewGetVaultToShow(ports.Lookup, ports.Vaults, ports.Last).Execute(opening.Named)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	window := buildWindow(ctx, a, entries, ports, db, opening)
	if err := startWindow(ctx, window.wiring, first); err != nil {
		return nil, errors.Join(err, window.closeEmbedder(), db.Close())
	}
	return window.wiring, nil
}

func buildWindow(
	ctx context.Context, a pluginsWiring, entries *logsusecase.Entries, ports registry.Ports, db *Index, opening WindowOpening,
) windowBuilt {

	// One task list serves every operation and the window that displays it.
	// A request is one task regardless of how it was issued. The list is created
	// before any component that reports into it.
	tasks := run.NewTasks()
	signalFailureClears(a)
	reportDependencies(a, tasks)
	setPluginWork(a, tasks, &run.Lock{})
	setModelCalls(a, run.NewConnectionNoticeTasks(tasks).Execute)

	// Opened once, for as long as the window is. A local model is fetched and
	// compiled behind this, so the window is drawn while it arrives. A source that cannot describe itself
	// yet is opened all the same: the search item of its plugin shows why, and the source joins search and
	// indexing once a configuration of its plugin lets it describe itself.
	embedder, asking, closeEmbedder, why := searchSources(a.search, ctx)
	embedding := indexing.NewEmbedding(embedder, embedder, db.ChunkVectors())
	state := editor.NewState(tasks, opening.Frame, func(event string) { emitPluginEvent(a, event) })
	state.SetIndexing(embedding)
	generations := newIndexGenerations(a.vault.Clock)
	indexer := newIndexWorker(a, db, state, embedding, emitIndexChanged(generations, state.Events), opening.Out)
	state.Writes.NotifyNoteWritten = indexer.WakeNotes

	// A note saved by the user is refreshed in the index before the save returns,
	// so the vault reflects its content without waiting for the watcher.
	current := &vault.CurrentRefresh{Fallback: func() vault.Refreshing {
		v := a.vault
		v.CutNotes = cutNotes(a.search)
		return vaultMakeRefresh(v, db, db.Notes())
	}}
	cases := newWindowUseCases(a, ports, db, state, current)

	session := newWindowSession(a, state, ctx, windowParts{
		indexer: indexer, embedding: embedding, passes: startVault(a, db, current),
		recordOpen: cases.Vaults.RecordOpen, importFiles: cases.Vaults.Tree.Import, out: opening.Out,
		embedderError: why, closeEmbedder: closeEmbedder, closeIndex: db.Close, frontmatter: db.FrontmatterChanges(),
		startFeed: getFeedStart(a, db),
	})
	cases.ShowTab = session.tools.showTab
	cases.OpenPlugin = session.cases.openPlugin
	inst := editor.NewInstallation(state, editor.Lifecycle{
		Start: session.cases.start, Show: session.cases.show,
		Flush: session.cases.flush, Close: session.cases.close,
	}, session.tools.importFiles, getMediaType(a.vault), opening.Out)

	kinds := newTabKinds(a, db)
	return windowBuilt{
		wiring: &windowWiring{
			logEntries: entries, a: a, ports: ports, db: db, embedder: embedder, asking: asking, inst: inst, state: state,
			kinds: kinds, server: newFileServer(a, state, cases), switching: session.userSwitch, cases: cases,
			tasks: tasks, generations: generations,
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
			Writes: none.OpenGate{},
		},
	)
	moves := vaultVaults(a.vault, ports, db, notes.Place.Move)
	listing := vaultListing(a.vault, ports)
	return WindowUseCases{Notes: &notes, Queries: &queries, PluginNotes: &pluginNotes, Listing: &listing, Vaults: &moves}
}

func newTabKinds(a pluginsWiring, db *Index) tabKinds {
	files := file.NewGetFileKinds(vaultReaders(a.vault), db.NoteQueries())
	chooser := getTabKindChooser(a)
	return tabKinds{files: files, chooser: chooser, list: file.NewListTabKinds(db.NoteQueries(), chooser)}
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
	mountCommand(w)
	mountConnection(w)
	mountConnectionCall(w)
	mountLicense(w)
	mountText(w)
	mountPlugin(w)
	mountHealth(w)
	mountLogs(w)
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
