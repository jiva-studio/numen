package container

import (
	"context"
	"io"

	windowagent "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/agent"
	windowcatalog "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/catalog"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	mediawindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/media"
	notewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/note"
	windowplugin "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/plugin"
	windowsearch "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/search"
	windowsettings "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/settings"
	windowtext "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/text"
	windowvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vaults"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/windowstate"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/workspace"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func mountCatalog(w *windowWiring) {
	if plugins := getPluginCatalog(plugins(w.cfg)); plugins != nil {
		handler := windowcatalog.NewHandler(plugins)
		path, served := windowcatalog.NewRoute(handler, w.inst.State.Gate.CountQuestions())
		w.inst.Mount(editor.NewRoute(path, served, handler))
	}
}

func mountMedia(w *windowWiring) {
	state := w.inst.State
	resolve := file.NewResolveAddress(file.NewStatFile(vaultReaders(vaultOf(w.cfg))), openPluginFile(vaultOf(w.cfg)))
	handler := mediawindow.NewHandler(state, state, resolve, getMediaType(vaultOf(w.cfg)))
	w.inst.Mount(editor.NewMediaRoute(state, handler))
}

func mountCommand(w *windowWiring) {
	var versions command.VersionItems
	if menu := openVersionItems(plugins(w.cfg)); menu != nil {
		versions = menu
	}
	handler := command.NewHandler(w.inst.State, getPluginCommands(plugins(w.cfg)), versions)
	w.inst.Mount(editor.NewRoute(command.NewRoute(w.inst.State.Gate.CountQuestions(), handler)))
}

func mountText(w *windowWiring) {
	handler := windowtext.NewHandler(w.inst.State, nil)
	if text := openFileText(vaultOf(w.cfg)); text != nil {
		handler = windowtext.NewHandler(w.inst.State, text)
	}
	path, served := windowtext.NewRoute(handler, w.inst.State.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func mountVaults(w *windowWiring) {
	state := w.inst.State
	list := w.vaults
	handler := vaults.NewHandler(state, vaults.Uses{
		List: list.Registry.List, Find: list.Registry.Find, FolderCheck: list.Registry.FolderCheck,
		Add: list.Change.Add, Rename: list.Change.Rename, Remove: list.Change.Remove,
	})
	w.inst.OnFolderDialog(func(dialog editor.PathDialog) {
		handler.SetFolderChooser(picker.NewChooseFolder(dialog))
	})
	path, served := vaults.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func mountSearch(w *windowWiring) {
	state := w.inst.State
	handler := windowsearch.NewHandler(state)
	handler.GetNameMatches = search.NewGetNameMatches(w.db.NoteQueries.NameMatches)
	handler.GetTitles = query.NewGetTitles(w.db.NoteQueries.NoteRefs)
	handler.GetTypes = query.NewGetTypes(w.db.NoteQueries.NoteTypes)
	handler.ListTabKinds = w.files.ListTabKinds
	finds := newSearch(searchOf(w.cfg), w.db, w.asking, state.ReportWordsAlone)
	handler.Finds = &finds
	setPluginSearch(searchOf(w.cfg), w.db, w.embedder, state.ReportFloorError)
	path, served := windowsearch.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func mountFile(w *windowWiring) {
	state := w.inst.State
	readers := vaultReaders(vaultOf(w.cfg))
	folders := filewindow.NewFolders(
		editor.NewSession(state), state.Events,
		file.NewGetFolderEntries(readers, w.db.NoteQueries.NoteTypes),
		file.NewGetFileKinds(readers, w.db.NoteQueries.NoteTypes),
		openChooseTabKind(plugins(w.cfg)),
		file.NewCreateFolder(folderWriters(vaultOf(w.cfg))),
	)
	changes := filewindow.NewChanges(editor.NewSession(state), state.Writes, state.Notices, &w.vaults.Tree.Move, &w.notes.Place.Remove, &w.notes.Place.EmptyTrash)
	handler := filewindow.NewHandler(folders, changes)
	w.files = handler
	w.server = editor.NewFileServer(
		state, &w.vaults.Registry.Find, file.NewOpenFile(readers), openPluginFile(vaultOf(w.cfg)), getMediaType(vaultOf(w.cfg)),
	)
	w.inst.Mount(editor.NewFileRoute(state, handler))
}

func mountWindowState(w *windowWiring) {
	state := w.inst.State
	path, served := windowstate.NewRoute(state.Window, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, state.Window))
}

func mountWorkspace(w *windowWiring) error {
	state := w.inst.State
	links, err := workspace.NewLinks()
	if err != nil {
		return err
	}
	store := w.db.Stores.OpenTabs
	handler := workspace.NewHandler(
		state, state.Notices, state, vault.NewWriteOpenTabs(store), vault.NewReadOpenTabs(store), links,
	)
	path, served := workspace.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
	return nil
}

func mountNote(w *windowWiring) {
	state := w.inst.State
	notes := w.notes
	addresses := notewindow.NewAddresses(state, notes.Refer.GetLinkAddress)
	session := editor.NewSession(state)
	prose := notewindow.NewProse(session, addresses, &notes.Edit.Read, &notes.Edit.Write, &notes.Edit.Create, &notes.Place.Rename)
	outline := notewindow.NewOutline(
		state, w.files.ListTabKinds,
		query.NewGetOpeningNote(w.db.NoteQueries.OpeningNote),
		query.NewGetTypes(w.db.NoteQueries.NoteTypes),
		query.NewGetHeadings(w.db.NoteQueries.Headings),
		&notes.Browse.Neighborhood,
	)
	links := notewindow.NewLinks(
		session, addresses,
		query.NewResolveLinks(w.db.NoteQueries.Links), &notes.Edit.Linking, &notes.Connect.CreateInRelation, &notes.Refer.Backlinks,
	)
	connections := notewindow.NewConnections(
		session, &notes.Browse.Connection, &notes.Connect.SaveConnection, &notes.Connect.RemoveSide, &notes.Connect.RemoveConnection, &notes.Connect.Join,
	)
	handler := notewindow.NewHandler(state.Notices, prose, outline, links, connections)
	w.inst.Mount(editor.NewNoteRoute(state, handler))
}

func mountAgent(w *windowWiring) {
	state := w.inst.State
	handler := &windowagent.Handler{
		GetRoot:        func() string { return state.GetCurrentVault().Path },
		GetVault:       func() domain.VaultID { return state.GetCurrentVault().ID },
		GetUnreachable: state.Unreachable.Why,
	}
	if taking := getToolProvider(plugins(w.cfg)); taking != nil {
		handler.Start = agent.NewStartTask(taking)
		handler.End = agent.NewEndConversation(taking)
	}
	path, served := windowagent.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

// GetSettingsAgentHandler is the window handler of the agent tasks, over the agent of the settings; it
// has no use case where no agent is configured.
func GetSettingsAgentHandler(c Config) *windowagent.Handler {
	handler := &windowagent.Handler{}
	if taking := getSettingsAgent(plugins(c)); taking != nil {
		handler.Start = agent.NewStartTask(taking)
		handler.End = agent.NewEndConversation(taking)
	}
	return handler
}

func openGetSettings(c Config, provider agentport.AgentProvider) *settings.Get {
	return settings.NewGet(readSettings(settingsOf(c)), modelsForProvider(c, provider), settingsPlugins(settingsOf(c)), func() []plugin.Setting { return listAppSettings(settingsOf(c)) })
}

func mountSettings(w *windowWiring) {
	state := w.inst.State
	handler := &windowsettings.Handler{}
	handler.Get = openGetSettings(w.cfg, pluginsAgentProvider(plugins(w.cfg)))
	handler.Values.Write = openWriteSettings(settingsOf(w.cfg))
	handler.Watch = openWatchSettings(settingsOf(w.cfg))
	handler.Retry = openRetryPlugin(settingsOf(w.cfg), getPluginRetrier(plugins(w.cfg)))
	if secrets := openSecrets(plugins(w.cfg)); secrets != nil {
		handler.ListSecrets = secrets.List
		handler.Create, handler.Rename, handler.SecretItems.Write = secrets.Items.Create, secrets.Items.Rename, secrets.Items.Write
		handler.Delete, handler.Move = secrets.Items.Delete, secrets.Items.Move
		handler.RenameGroup, handler.DeleteGroup = secrets.Groups.Rename, secrets.Groups.Delete
		handler.Allow, handler.Deny, handler.ListStates = secrets.Access.Allow, secrets.Access.Deny, secrets.Access.ListStates
	}
	if data := openPluginData(vaultOf(w.cfg)); data != nil {
		handler.Data, handler.Vaults = data, state
	}
	w.inst.OnFolderDialog(func(dialog editor.PathDialog) {
		handler.ChoosePath = openChoosePluginPath(settingsOf(w.cfg), dialog)
	})
	path, served := windowsettings.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func mountPlugin(w *windowWiring) {
	w.inst.SetPlugins(getPluginServices(plugins(w.cfg)), func(id string) (string, bool) { return getPluginDir(plugins(w.cfg), id) })
	w.inst.Mount(editor.NewRoute(windowplugin.NewRoute(w.inst.State.Gate.CountQuestions())))
}

// shareEditorWindow gives the plugins the window: it focuses a note and the passages of a file
// in it and returns the tabs it last reported. The window reads the vault of its state on each call,
// so a vault opened later in the window is served without a new binding.
func shareEditorWindow(c Config, db *vault.Index, opened *editor.Installation) {
	view := GetWindowView(opened)
	focusPassages := file.NewFocusPassages(vaultReaders(vaultOf(c)), view)
	setPluginWindow(plugins(c), windowsession.NewPluginWindow(
		&opened.State.Runtime,
		vaultFocusNote(vaultOf(c), db.NoteQueries.NoteRefs, view),
		windowsession.PassagesFocusFunc(func(ctx context.Context, v domain.Vault, path string, passages []domain.ByteRange) error {
			return focusPassages.Execute(ctx, v, file.FocusPassagesInput{Path: path, Passages: passages})
		}),
		opened.ShowTabs(),
	))
}

// shareVaultList gives the plugins the list of vaults of the window and the vault it
// shows. It uses the vault use cases of the window, so a plugin and the user see the same errors.
func shareVaultList(c Config, opened *editor.Installation) {
	scenarios := opened.Vaults()
	setPluginVaultList(plugins(c), windowsession.NewPluginVaults(&opened.State.Runtime, windowsession.VaultOps{
		Edit:    windowsession.VaultEdits{Add: scenarios.Change.Add, Rename: scenarios.Change.Rename, Remove: scenarios.Change.Remove},
		Find:    scenarios.Registry.Find,
		Missing: scenarios.Registry.FolderCheck,
		Open:    opened,
	}))
}

func mountVault(w *windowWiring) {
	state := w.inst.State
	handler := windowvault.NewHandler(state, state.Notices)
	path, served := windowvault.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func getReadVault(cfg Config, db *vault.Index) indexing.OpenReading {
	return func(
		ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
	) (indexing.ReadWholeVault, error) {
		request := cfg
		request.ShouldRebuildIndex = rebuild
		return ReadWholeVault(request, ctx, db, embedder, v)
	}
}

func newIndexWorker(w *windowWiring, state *editor.State, embedding *indexing.Embedding, out io.Writer) *indexing.Worker {
	return indexing.NewWorker(indexing.Config{
		Reading:  getReadVault(w.cfg, w.db),
		Embedder: embedding.GetEmbedder(),
		VectorCounts: func(ctx context.Context, v domain.Vault) (held, embedded int64, err error) {
			if !embedding.HasProgress() {
				return 0, 0, indexing.ErrNoProgress
			}
			name, recipe := embedding.Describe()
			state.Runtime.RecordModel(name, recipe)
			return embedding.CountProgress(ctx, v.ID, recipe)
		},
		VectorsReady: state.Events.EmitVectorsReady,
		Events:       windowsession.NewIndexEvents(&state.Runtime, state.Window, out),
		Quiet:        indexing.QuietPeriod,
		After:        clock.System{}.After,
	})
}

func newIndexSession(worker *indexing.Worker) func() windowsession.IndexSession {
	return func() windowsession.IndexSession {
		session := worker.NewSession()
		return windowsession.IndexSession{
			Run: func(ctx context.Context, v domain.Vault, rebuild bool, read windowsession.ReadNotes) {
				session.Run(ctx, v, rebuild, read)
			},
			ReportVersionsChanged: session.ReportVersionsChanged,
		}
	}
}

func getTextVersions(c Config) windowsession.TextVersions {
	if versions := vaultOf(c).TextVersions; versions != nil {
		return versions.Changes
	}
	return nil
}

func getTriggerSource(c Config) windowsession.TriggerSource {
	return windowsession.TriggerSource{
		Open: func() windowsession.TriggerRunner {
			if triggers := openTriggers(plugins(c)); triggers != nil {
				return triggers
			}
			return nil
		},
		Watch: func() (<-chan struct{}, func()) { return watchPluginChanges(plugins(c)) },
	}
}

func startVault(c Config, db *vault.Index, current *vault.CurrentRefresh) windowsession.VaultPassesFunc {
	return func(
		ctx context.Context, v domain.Vault, rebuild bool, report windowsession.Reporters,
	) windowsession.Started {
		request := c
		request.ShouldRebuildIndex = rebuild

		opening := vaultOpener(vaultOf(request), db)
		opening.ShouldRebuild = rebuild
		opening.ErrorHandler = report.Failed
		if report.Changed != nil {
			opening.Told = func(m vault.VaultChanges) { report.Changed(m.Paths, m.Assets, m.ShouldReload) }
		}

		open := opening.Begin(ctx, v)
		refresh := opening.GetRefresh()
		return windowsession.Started{
			Read:      open.Read,
			Run:       open.Run,
			Unwatched: open.GetUnwatchedReason(),
			Publish:   func() { current.Publish(refresh) },
			Retire:    current.Retire,
		}
	}
}
