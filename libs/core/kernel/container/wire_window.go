package container

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
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
	ports, err := c.registry()
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return c.openWindow(exec, prefs, endpoints, res, ctx, windowStartup{opening: opening, ports: ports, db: db})
}

func (c Config) openWindow(exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, ctx context.Context, startup windowStartup) (*OpenedWindow, error) {
	exec = getPluginExecution(exec)
	db := startup.db
	install := newPluginInstall(c, endpoints, res)
	var versions *TextVersions
	if startup.isEditor {
		base := vaultBase(c, getRunningExecution(exec, c.SafeMode), install.Endpoints, res, nil)
		versions = newTextVersions(base, db)
	}
	settingsDeps, searchDeps, vaultDeps := buildVaultSearch(c, exec, prefs, install.Endpoints, res, versions)
	ports := startup.ports
	first, err := registry.NewGetVaultToShow(ports.Lookup, ports.Vaults, ports.Last).Execute(startup.opening.Named)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	plugins := openWindowPlugins(install, exec, settingsDeps, vaultDeps, db)
	opened, closeEmbedder := buildWindow(startup, ctx, plugins, searchDeps, vaultDeps, getLogEntries(res))
	mountServices(opened, install, exec, settingsDeps, searchDeps, vaultDeps)
	if err := opened.inst.Start(ctx, first, opened.server); err != nil {
		return nil, errors.Join(err, closeEmbedder(), db.Close())
	}

	if startup.isEditor {
		return connectEditor(install, exec, vaultDeps, startup.ports, db, opened)
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

type windowStartup struct {
	isEditor bool
	opening  WindowOpening
	ports    registry.Ports
	db       *Index
}

type windowPlugins struct {
	tasks     *run.Tasks
	events    pluginport.PluginEventEmitter
	vault     windowsession.PluginVaultPath
	triggers  windowsession.TriggerSource
	startFeed func(context.Context, domain.Vault) func()
	kinds     tabKinds
}

func openWindowPlugins(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring, db *Index) windowPlugins {
	tasks := run.NewTasks()
	signalFailureClears(exec, settingsDeps)
	reportDependencies(exec.Watch, tasks, vaultDeps.RecordLog)
	pluginsOpenPluginWork(install, exec, tasks, &run.Lock{})
	setModelCalls(install, exec, settingsDeps, vaultDeps, run.NewConnectionNoticeTasks(tasks).Execute)
	return windowPlugins{
		tasks: tasks, events: exec.Runtime.PluginEventEmitter, vault: exec.Runtime.PluginBinding,
		triggers:  getTriggerSource(install, exec, settingsDeps, vaultDeps, db.FrontmatterChanges()),
		startFeed: getFeedStart(exec, settingsDeps, vaultDeps, db), kinds: newTabKinds(exec, settingsDeps, vaultDeps, db),
	}
}

func buildWindow(
	s windowStartup, ctx context.Context, plugins windowPlugins, searchDeps searchWiring, vaultDeps vaultWiring, entries *logsusecase.Entries,
) (*windowWiring, func() error) {
	db, opening, ports := s.db, s.opening, s.ports
	tasks := plugins.tasks
	// Opened once, for as long as the window is. A local model is fetched and
	// compiled behind this, so the window is drawn while it arrives. A source that cannot describe itself
	// yet is opened all the same: the search item of its plugin shows why, and the source joins search and
	// indexing once a configuration of its plugin lets it describe itself.
	embedder, asking, closeEmbedder, why := searchSources(searchDeps, ctx)
	embedding := indexing.NewEmbedding(embedder, embedder, db.ChunkVectors())
	state := editor.NewState(tasks, opening.Frame, func(event string) { emitPluginEvent(plugins.events, event) })
	state.SetIndexing(embedding)
	generations := newIndexGenerations(vaultDeps.Clock)
	indexer := newIndexWorker(s, searchDeps, vaultDeps, state, embedding, emitIndexChanged(generations, state.Events))
	state.Writes.NotifyNoteWritten = indexer.WakeNotes

	// A note saved by the user is refreshed in the index before the save returns,
	// so the vault reflects its content without waiting for the watcher.
	current := &vault.CurrentRefresh{Fallback: func() vault.Refreshing {
		v := vaultDeps
		v.CutNotes = cutNotes(searchDeps)
		return vaultMakeRefresh(v, db, db.Notes())
	}}
	cases := newWindowUseCases(vaultDeps, ports, db, state, current)

	session := newWindowSession(plugins.vault, vaultDeps, state, ctx, windowParts{
		triggers: plugins.triggers, shouldRebuild: searchDeps.ShouldRebuildIndex,
		indexer: indexer, embedding: embedding, passes: startVault(vaultDeps, db, current),
		recordOpen: cases.Vaults.RecordOpen, importFiles: cases.Vaults.Tree.Import, out: opening.Out,
		embedderError: why, closeEmbedder: closeEmbedder, closeIndex: db.Close,
		startFeed: plugins.startFeed,
	})
	cases.ShowTab = session.tools.showTab
	cases.OpenPlugin = session.cases.openPlugin
	inst := editor.NewInstallation(state, editor.Lifecycle{
		Start: session.cases.start, Show: session.cases.show,
		Flush: session.cases.flush, Close: session.cases.close,
	}, session.tools.importFiles, getMediaType(vaultDeps), opening.Out)

	window := &windowWiring{
		logEntries: entries, ports: ports, db: db, embedder: embedder, asking: asking, inst: inst, state: state,
		kinds: plugins.kinds, server: newFileServer(vaultDeps, state, cases), switching: session.userSwitch, cases: cases,
		tasks: tasks, generations: generations,
	}
	return window, closeEmbedder
}

func newWindowUseCases(
	vaultDeps vaultWiring, ports registry.Ports, db *Index, state *editor.State, current *vault.CurrentRefresh,
) WindowUseCases {
	notes := vaultNotes(
		vaultDeps, db, vault.NewUpdateIndex(current.Get).Execute,
		noteViews{
			Moves:  note.NewTellMoveInShownVault(state.GetShownVault, windowView(state)),
			Writes: state.Writes,
			Follow: state.Writes,
		},
	)
	queries := vaultQueries(db)
	pluginNotes := vaultNotes(
		vaultDeps, db, vault.NewUpdateIndex(current.Get).Execute,
		noteViews{
			Moves:  note.NewTellMoveInShownVault(state.GetShownVault, windowView(state)),
			Writes: none.OpenGate{},
		},
	)
	moves := vaultVaults(vaultDeps, ports, db, notes.Place.Move)
	listing := vaultListing(vaultDeps, ports)
	return WindowUseCases{Notes: &notes, Queries: &queries, PluginNotes: &pluginNotes, Listing: &listing, Vaults: &moves}
}

func newTabKinds(exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring, db *Index) tabKinds {
	files := file.NewGetFileKinds(vaultReaders(vaultDeps), db.NoteQueries())
	chooser := openChooseTabKind(exec, settingsDeps)
	return tabKinds{files: files, chooser: chooser, list: file.NewListTabKinds(db.NoteQueries(), chooser)}
}

func newFileServer(vaultDeps vaultWiring, state *editor.State, cases WindowUseCases) *editor.FileServer {
	return editor.NewFileServer(
		state,
		file.NewOpenVaultFile(
			&cases.Listing.Find, file.NewOpenFile(vaultReaders(vaultDeps)), openPluginFile(vaultDeps), getMediaType(vaultDeps),
		),
	)
}

func mountServices(w *windowWiring, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, searchDeps searchWiring, vaultDeps vaultWiring) {
	mountVault(w)
	mountWorkspace(w)
	mountWindowState(w)
	mountFile(w, vaultDeps)
	mountNote(w)
	mountSearch(w, install, exec, settingsDeps, searchDeps, vaultDeps)
	mountVaults(w)
	mountSettings(w, exec, settingsDeps, vaultDeps)
	mountMedia(w, vaultDeps)
	mountCommand(w, install, exec, settingsDeps, vaultDeps)
	mountConnection(w, install, exec, settingsDeps, vaultDeps)
	mountConnectionCall(w, install, exec, settingsDeps, vaultDeps)
	mountLicense(w, install, settingsDeps, vaultDeps)
	mountText(w, vaultDeps)
	mountPlugin(w, install, exec, settingsDeps, vaultDeps)
	mountHealth(w, exec)
	mountLogs(w, install, settingsDeps, vaultDeps)
	mountCatalog(w, install, settingsDeps)
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
