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
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vaultlist"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func mountCatalog(w *windowWiring) {
	if plugins := getPluginCatalog(plugins(w.cfg)); plugins != nil {
		handler := windowcatalog.NewHandler(plugins)
		path, served := windowcatalog.NewRoute(handler, w.state.Gate.CountQuestions())
		w.inst.Mount(editor.NewRoute(path, served, handler))
	}
}

func mountMedia(w *windowWiring) {
	state := w.state
	resolve := file.NewResolveAddress(file.NewStatFile(vaultReaders(vaultOf(w.cfg))), openPluginFile(vaultOf(w.cfg)), getMediaType(vaultOf(w.cfg)))
	handler := mediawindow.NewHandler(state, state, resolve)
	w.inst.Mount(editor.NewMediaRoute(state, handler))
}

func mountCommand(w *windowWiring) {
	p := plugins(w.cfg)
	var versions plugincommand.VersionItems
	if menu := openVersionItems(p); menu != nil {
		versions = menu
	}
	commands := newPluginCommands(p, p.PluginHost, versions)
	handler := command.NewHandler(w.state, commands.Listing, commands.RunCommand, commands.ListItems)
	w.inst.Mount(editor.NewRoute(command.NewRoute(w.state.Gate.CountQuestions(), handler)))
}

func mountText(w *windowWiring) {
	handler := windowtext.NewHandler(w.state, nil)
	if text := openFileText(vaultOf(w.cfg)); text != nil {
		handler = windowtext.NewHandler(w.state, text)
	}
	path, served := windowtext.NewRoute(handler, w.state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func mountVaults(w *windowWiring) {
	state := w.state
	registry, change := w.cases.Listing, w.cases.Vaults.Change
	handler := vaults.NewHandler(
		state,
		vaultlist.NewListVaults(registry.List, registry.FolderCheck),
		vaultlist.NewAddVault(change.Add, registry.FolderCheck),
		vaultlist.NewRenameVault(registry.Find, change.Rename, registry.FolderCheck),
		vaultlist.NewOpenVault(registry.Find, state),
		change.Remove,
	)
	w.inst.OnFolderDialog(func(dialog editor.PathDialog) {
		handler.SetFolderChooser(picker.NewChooseFolder(dialog))
	})
	path, served := vaults.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func mountSearch(w *windowWiring) {
	state := w.state
	finds := newSearch(searchOf(w.cfg), w.db, w.asking, state.ReportWordsAlone)
	handler := windowsearch.NewHandler(
		state,
		search.NewFindNames(w.db.NoteQueries(), w.tabKinds),
		search.NewFindPassages(finds, w.db.NoteQueries(), w.db.NoteQueries(), w.tabKinds),
	)
	setPluginSearch(searchOf(w.cfg), w.db, w.embedder, state.ReportFloorError)
	path, served := windowsearch.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func getTabKindChooser(c pluginsWiring) file.TabKindChooser {
	if choose := openChooseTabKind(c); choose != nil {
		return choose
	}
	return file.NoTabKinds{}
}

func mountFile(w *windowWiring) {
	state := w.state
	readers := vaultReaders(vaultOf(w.cfg))
	fileKinds := file.NewGetFileKinds(readers, w.db.NoteQueries())
	chooser := getTabKindChooser(plugins(w.cfg))
	w.tabKinds = file.NewListTabKinds(fileKinds, chooser)
	folders := filewindow.NewFolders(
		state, state.Events,
		file.NewListFolder(file.NewGetFolderEntries(readers, w.db.NoteQueries()), w.tabKinds),
		file.NewListFileKinds(fileKinds, chooser),
		file.NewCreateFolder(folderWriters(vaultOf(w.cfg)), state.Writes),
	)
	moveFile := vault.NewMoveFile(&w.cases.Vaults.Tree.Move, state.Writes, state.Writes)
	changes := filewindow.NewChanges(
		state, state.Notices, &moveFile, &w.cases.Notes.Place.Remove, &w.cases.Notes.Place.EmptyTrash, state.Writes,
	)
	handler := filewindow.NewHandler(folders, changes)
	w.server = editor.NewFileServer(
		state,
		file.NewOpenVaultFile(
			&w.cases.Listing.Find, file.NewOpenFile(readers), openPluginFile(vaultOf(w.cfg)), getMediaType(vaultOf(w.cfg)),
		),
	)
	w.inst.Mount(editor.NewFileRoute(state, handler))
}

func mountWindowState(w *windowWiring) {
	state := w.state
	path, served := windowstate.NewRoute(state.Window, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, state.Window))
}

func mountWorkspace(w *windowWiring) error {
	state := w.state
	links, err := workspace.NewLinks()
	if err != nil {
		return err
	}
	store := w.db.OpenTabs()
	handler := workspace.NewHandler(
		state, state.Notices, state, vault.NewWriteOpenTabs(store), vault.NewReadOpenTabs(store), links,
	)
	path, served := workspace.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
	return nil
}

func mountNote(w *windowWiring) {
	state := w.state
	notes, queries := w.cases.Notes, w.cases.Queries
	writes := state.Writes
	prose := notewindow.NewProse(
		state, &notes.Edit.Read, &notes.Edit.Write, &notes.Edit.Create, &notes.Place.Rename, writes,
	)
	outlining := notewindow.NewOutline(
		state,
		query.NewGetOpeningNote(w.db.NoteQueries()),
		query.NewGetNeighborhoodView(w.db.NoteQueries(), w.db.NoteQueries(), w.db.NoteQueries(), w.tabKinds),
		query.NewListHeadings(w.db.NoteQueries()),
	)
	writeLink := note.NewWriteLink(notes.Edit.Linking, w.db.NoteQueries(), writes)
	links := notewindow.NewLinks(
		state, query.NewResolveLinks(w.db.NoteQueries()), &writeLink, &notes.Connect.CreateInRelation, &queries.Refer.Backlinks, writes,
	)
	connections := notewindow.NewConnections(
		state, &queries.Browse.Connection, &notes.Connect.SaveConnection, &notes.Connect.RemoveSide, &notes.Connect.RemoveConnection, &notes.Connect.Join,
	)
	handler := notewindow.NewHandler(state.Notices, prose, outlining, links, connections)
	w.inst.Mount(editor.NewNoteRoute(state, handler))
}

func mountAgent(w *windowWiring) {
	state := w.state
	handler := &windowagent.Handler{
		GetRoot:  func() string { return state.GetCurrentVault().Path },
		GetVault: func() domain.VaultID { return state.GetCurrentVault().ID },
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
	state := w.state
	var secretLister settings.SecretLister
	items, groups := &windowsettings.SecretItems{}, &windowsettings.SecretGroups{}
	if secrets := openSecrets(plugins(w.cfg)); secrets != nil {
		secretLister = secrets.List.ListForSettings
		items = windowsettings.NewSecretItems(
			secrets.Items.Create, secrets.Items.Rename, secrets.Items.Write, secrets.Items.Delete, secrets.Items.Move,
		)
		groups = windowsettings.NewSecretGroups(
			secrets.Groups.Rename, secrets.Groups.Delete, secrets.Access.Allow, secrets.Access.Deny, secrets.Access.ListStates,
		)
	}
	retry := openRetryPlugin(settingsOf(w.cfg), getPluginRetrier(plugins(w.cfg)))
	folders := windowsettings.NewPluginFolders(retry, nil, nil)
	if data := openPluginData(vaultOf(w.cfg)); data != nil {
		folders = windowsettings.NewPluginFolders(retry, data, state)
	}
	values := windowsettings.NewValues(
		settings.NewGetWithSecrets(openGetSettings(w.cfg, pluginsAgentProvider(plugins(w.cfg))), secretLister),
		openWriteSettings(settingsOf(w.cfg)), openWatchSettings(settingsOf(w.cfg)), nil,
	)
	handler := windowsettings.NewHandler(values, folders, items, groups)
	w.inst.OnFolderDialog(func(dialog editor.PathDialog) {
		handler.SetPathChooser(openChoosePluginPath(settingsOf(w.cfg), dialog))
	})
	path, served := windowsettings.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func mountPlugin(w *windowWiring) {
	w.inst.SetPlugins(getPluginServices(plugins(w.cfg)), func(id string) (string, bool) { return getPluginDir(plugins(w.cfg), id) })
	w.inst.Mount(editor.NewRoute(windowplugin.NewRoute(w.state.Gate.CountQuestions())))
}

func shareEditorWindow(c Config, db *Index, opened *windowWiring) {
	view := windowView(opened.state)
	focusPassages := file.NewFocusPassages(vaultReaders(vaultOf(c)), view)
	setPluginWindow(plugins(c), windowsession.NewPluginWindow(
		&opened.state.Runtime,
		&opened.state.Tabs,
		vaultFocusNote(vaultOf(c), db.NoteQueries(), view),
		windowsession.PassagesFocusFunc(func(ctx context.Context, v domain.Vault, path string, passages []domain.ByteRange) error {
			return focusPassages.Execute(ctx, v, file.FocusPassagesInput{Path: path, Passages: passages})
		}),
		opened.cases.ShowTab,
	))
}

func shareVaultList(c Config, opened *windowWiring) {
	scenarios := opened.cases.Vaults
	setPluginVaultList(plugins(c), windowsession.NewPluginVaults(&opened.state.Runtime, windowsession.VaultOps{
		Edit:    windowsession.VaultEdits{Add: scenarios.Change.Add, Rename: scenarios.Change.Rename, Remove: scenarios.Change.Remove},
		Find:    opened.cases.Listing.Find,
		Missing: opened.cases.Listing.FolderCheck,
		Open:    opened.inst,
	}))
}

func mountVault(w *windowWiring) {
	state := w.state
	handler := windowvault.NewHandler(state, state.Notices)
	path, served := windowvault.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func getReadVault(cfg Config, db *Index) indexing.OpenReading {
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
		Reading:      getReadVault(w.cfg, w.db),
		Embedder:     embedding.GetEmbedder(),
		VectorCounts: indexing.NewVectorCounts(embedding, &state.Runtime).Execute,
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

func startVault(c Config, db *Index, current *vault.CurrentRefresh) windowsession.VaultPassesFunc {
	return func(
		ctx context.Context, v domain.Vault, rebuild bool, report windowsession.Reporters,
	) windowsession.Started {
		request := c
		request.ShouldRebuildIndex = rebuild

		opening := vaultOpener(vaultOf(request), db)
		opening.ShouldRebuild = rebuild
		var changed func(vault.VaultChanges)
		if report.Changed != nil {
			changed = func(m vault.VaultChanges) { report.Changed(m.Paths, m.Assets, m.ShouldReload) }
		}

		open := opening.Begin(ctx, v, changed, report.Failed)
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
