package container

import (
	"context"
	"errors"
	"io"

	pluginvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexgeneration"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// tabKinds are the use cases that name the kind of a file for the tabs of a window.
type tabKinds struct {
	files   *file.GetFileKinds
	chooser file.TabKindChooser
	list    *file.ListTabKinds
}

// OpenWindow opens the registry and the cache, builds the services of the editor window over
// them, and opens the vault named, or the one shown last when none is named.
func (c Config) OpenWindow(exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, ctx context.Context, opening WindowOpening) (*editor.Installation, error) {
	db, err := c.openIndex(res, ctx)
	if err != nil {
		return nil, err
	}
	ports, err := c.registry()
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return c.openWindow(exec, prefs, endpoints, res, ctx, windowStartup{opening: opening, ports: ports, db: db})
}

func (c Config) openWindow(exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, ctx context.Context, startup windowStartup) (*editor.Installation, error) {
	exec = getPluginExecution(exec)
	db := startup.db
	install := newPluginInstall(c, prefs, endpoints, res)
	versions := getWindowVersions(c, exec, res, startup)
	settingsDeps, searchDeps, vaultDeps := buildVaultSearch(c, exec, prefs, res, versions)
	ports := startup.ports
	first, err := registry.NewGetVaultToShow(ports.Lookup, ports.Vaults, ports.Last).Execute(startup.opening.Named)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	plugins := openWindowPlugins(install, exec, settingsDeps, vaultDeps, db, newSourceBinding(exec, c.SafeMode, install.Endpoints, res))
	opened, launch := buildWindow(startup, ctx, plugins, searchDeps, vaultDeps)
	search := newTextSearch(searchDeps, c.machine.ResolvedCompute.Workers)
	session := windowSearch{index: db, source: launch.source, kinds: plugins.kinds.list}
	status := editor.SearchTasks{Tasks: opened.page.Window}
	query := newFindSearchResults(openQueryPorts(install, exec, settingsDeps, vaultDeps), settingsDeps, search, db, launch.source, status.ReportFloorError)
	runs := newSearchRuns(plugins.tasks, launch.generations, query, exec.Runtime.Resolver, exec.Runtime.Manifests, vaultDeps.Clock)
	mountSearch(opened, session, search, install.Endpoints.Lender, vaultDeps.Clock, runs)
	mountServices(opened, install, res.restore, exec, settingsDeps, vaultDeps)
	mountLogs(opened, getLogEntries(res), install, settingsDeps, vaultDeps)
	if err := opened.Start(ctx, first, launch.server, launch.start, startup.opening.Out); err != nil {
		return nil, errors.Join(err, launch.closeSource(), db.Close())
	}
	return startup.connect(ctx, install, exec, vaultDeps, opened, launch.plugins)
}

func getWindowVersions(c Config, exec *PluginExecution, res *Resources, startup windowStartup) *textVersions {
	if !startup.isEditor {
		return nil
	}
	running := getRunningExecution(exec, c.SafeMode)
	return newTextVersions(vaultconfig.Options{ServiceDir: c.ServiceDir}, running.Runtime.Listing, GetClock(res), startup.db)
}

// WindowOpening names the vault to show, the output and the optional editor frame.
type WindowOpening struct {
	Named string
	Out   io.Writer
	Frame editor.Frame
}

type windowRoutes struct {
	*editor.Installation
	session *editor.Session
	page    *editor.PageAccess
}

type windowPluginAccess struct {
	writeWake indexing.WriteWake
	notes     hostvault.NoteChanges
	vaults    pluginvault.ListAccess
	showTab   windowsession.ShowTab
}

type windowLaunch struct {
	start       windowsession.StartWindow
	server      *editor.FileServer
	closeSource func() error
	plugins     windowPluginAccess
	source      sourceAccess
	generations *indexgeneration.Generations
}
type windowStartup struct {
	isEditor bool
	opening  WindowOpening
	ports    registry.Ports
	db       *vaultIndex
}

type windowPlugins struct {
	source    sourcebridge.Binding
	tasks     *run.Tasks
	events    pluginEventEmitter
	vault     windowsession.PluginVaultPath
	triggers  windowsession.TriggerSource
	startFeed func(context.Context, domain.Vault) func()
	kinds     tabKinds
}

func openWindowPlugins(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring, db *vaultIndex, source sourcebridge.Binding) windowPlugins {
	tasks := run.NewTasks()
	exec.Runtime.SetOnFailureCleared(func(string) { settingsDeps.PluginChanges.Signal() })
	reportDependencies(exec.Watch.dependencyReporter, tasks, vaultDeps.RecordLog)
	pluginsOpenPluginWork(install, exec, tasks)
	setModelCalls(install, exec, settingsDeps, vaultDeps, run.NewConnectionNoticeTasks(tasks).Execute)
	return windowPlugins{
		source: source,
		tasks:  tasks, events: exec.Runtime.pluginEventEmitter, vault: exec.Runtime.pluginBinding,
		triggers:  getTriggerSource(install, exec, settingsDeps, vaultDeps, db.FrontmatterChanges()),
		startFeed: getFeedStart(exec, settingsDeps, vaultDeps, db), kinds: newTabKinds(exec, settingsDeps, vaultDeps, db),
	}
}

func buildWindow(
	s windowStartup, ctx context.Context, plugins windowPlugins, searchDeps vaultIndexPlan, vaultDeps vaultWiring,
) (*windowRoutes, windowLaunch) {
	db, opening, ports := s.db, s.opening, s.ports
	tasks := plugins.tasks
	// Opened once, for as long as the window is. A local model is fetched and
	// compiled behind this, so the window is drawn while it arrives. A source that cannot describe itself
	// yet is opened all the same: the search item of its plugin shows why, and the source joins search and
	// indexing once a configuration of its plugin lets it describe itself.
	embedder, closeEmbedder, why := plugins.source.OpenWatching(ctx)
	source := getSourceAccess(embedder)
	embedding := indexing.NewEmbedding(source.searchSource, source.watcher, db.ChunkVectors())
	state := editor.NewSession(opening.Frame)
	page := editor.NewPageAccess(tasks, state)
	events := &editor.Events{Emit: plugins.events.EmitEvent}
	state.Runtime.RecordModel(embedding.Describe())
	generations := newIndexGenerations(vaultDeps.Clock)
	indexer := s.newIndexWorker(searchDeps, vaultDeps, &state.Runtime, page.Window, embedding, emitIndexChanged(generations, events))
	writeWake := indexing.WriteWake(indexer.WakeWrittenVault)

	// A note saved by the user is refreshed in the index before the save returns,
	// so the vault reflects its content without waiting for the watcher.
	current := &vault.CurrentRefresh{Fallback: func() vault.Refreshing {
		v := vaultDeps
		v.CutNotes = cutNotes(searchDeps.ChunkPolicy)
		return vaultMakeRefresh(v, db, db.Notes())
	}}
	notes, pluginNotes := newWindowNotes(vaultDeps, db, state, noteViews{Admission: page.Writes.AdmitWrite, Follow: page.Writes}, current)
	moves := vaultVaults(vaultDeps, ports, db)

	diag := windowsession.NewDiagnostics(opening.Out, page.Window)
	runner := windowsession.NewPassRunner(windowsession.PassRunnerConfig{
		Failures: &state.Runtime, Diag: diag, Closing: ctx.Done(), NewSession: newIndexSession(indexer),
		Reporter: windowsession.NewChangeReporter(&state.Notices.ChangeEvents, indexer, getTextVersions(vaultDeps.TextVersions), diag),
		Passes:   startVault(vaultDeps, db, current), Triggers: plugins.triggers, StartFeed: plugins.startFeed,
		Report: editor.ReportVault(vaultDeps.RecordLog),
	})
	parts := windowParts{
		shouldRebuild: searchDeps.ShouldRebuildIndex,
		indexer:       indexer, embedding: embedding,
		recordOpen: ports.Recorder, lookup: ports.Lookup, importFiles: moves.Tree.Import, diagnostics: diag,
		embedderError: why, closeEmbedder: closeEmbedder, closeIndex: db.Close,
	}
	session := newWindowSession(parts, plugins.vault, runner, vaultDeps, state, page)
	inst := editor.NewInstallation(page, editor.Lifecycle{
		Show:  session.cases.show,
		Flush: session.cases.flush, Close: session.cases.close,
	}, session.tools.importFiles, getMediaType())

	window := &windowRoutes{Installation: inst, session: state, page: page}
	mountVault(window)
	mountWorkspace(window, db)
	mountWindowState(window)
	mountFile(window, vaultDeps, db, plugins.kinds, &notes.Place)
	mountNote(window, &notes, db, plugins.kinds.list, writeWake)
	mountVaults(window, ports, vaultDeps.Readers, moves.Change, session.open)
	return window, windowLaunch{
		start:  session.cases.start,
		server: newFileServer(vaultDeps, &state.Runtime, page.Window, ports.Lookup), closeSource: closeEmbedder, source: source, generations: generations,
		plugins: windowPluginAccess{
			writeWake: writeWake, notes: pluginNotes, vaults: newPluginVaultList(&state.Runtime, moves.Change, ports.Lookup, vaultDeps.Readers, session.cases.openPlugin), showTab: session.tools.showTab,
		},
	}
}

func newWindowNotes(vaultDeps vaultWiring, db *vaultIndex, state *editor.Session, views noteViews, current *vault.CurrentRefresh) (noteUseCases, hostvault.NoteChanges) {
	update := vault.NewUpdateIndex(current.Get).Execute
	moves := note.NewTellMoveInShownVault(state.GetShownVault, windowView(state.Notices))
	views.Moves = moves
	notes := vaultNotes(vaultDeps, db, update, views)
	pluginNotes := vaultNotes(vaultDeps, db, update, noteViews{Moves: moves})
	return notes, getNoteChanges(pluginNotes)
}

func newTabKinds(exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring, db *vaultIndex) tabKinds {
	files := file.NewGetFileKinds(vaultDeps.Readers, db.NoteQueries())
	chooser := file.NewChooseTabKind(exec.Runtime.Manifests, exec.Watch.ContentTypeTable, getWhenConfig(settingsDeps, exec.Runtime.Listing))
	return tabKinds{files: files, chooser: chooser, list: file.NewListTabKinds(db.NoteQueries(), chooser)}
}

func newFileServer(vaultDeps vaultWiring, runtime *windowsession.Runtime, tasks windowport.Tasks, find vaultport.VaultLookup) *editor.FileServer {
	return editor.NewFileServer(
		runtime, tasks,
		file.NewOpenVaultFile(
			find, file.NewOpenFile(vaultDeps.Readers), openPluginFile(vaultDeps.Options), getMediaType(),
		),
	)
}

func mountServices(w *windowRoutes, install pluginInstall, pending *identity.PendingStore, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	mountPluginData(w, vaultDeps)
	mountSettings(w, settingsDeps)
	mountMedia(w, vaultDeps)
	mountCommand(w, install, exec, settingsDeps, vaultDeps)
	mountConnection(w, install, exec, settingsDeps, vaultDeps)
	mountConnectionCall(w, install, exec, settingsDeps, vaultDeps, vaultDeps.Clock)
	mountLicense(w, install, settingsDeps, vaultDeps)
	mountIdentity(w, install, vaultDeps, pending)
	mountText(w, vaultDeps)
	mountPlugin(w, install, exec, settingsDeps, vaultDeps)
	mountHealth(w, exec)
	mountCatalog(w, install, exec, settingsDeps)
}

func windowView(notices *editor.Notices) windowsession.View {
	return windowsession.NewView(&notices.Places, &notices.Edits, &notices.ChangeEvents)
}

// GetUnknownVaultName returns the name of the vault, named at start, that err says the registry does not
// hold.
func GetUnknownVaultName(err error) (string, bool) {
	unknown, ok := errors.AsType[*vaultdomain.UnknownError](err)
	if !ok {
		return "", false
	}
	return unknown.Name, true
}
