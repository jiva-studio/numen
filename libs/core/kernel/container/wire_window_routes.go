package container

import (
	"context"
	"io"

	indexnote "github.com/jiva-studio/numen/libs/core/kernel/adapter/index/note"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	windowcatalog "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/catalog"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/command"
	windowconnection "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/connectioncall"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	mediawindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/media"
	notewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/note"
	windowhealth "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/health"
	windowlogs "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/logs"
	windowplugin "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/plugin"
	windowsearch "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/search"
	windowsettings "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/settings"
	windowtext "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/text"
	windowvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vaults"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/windowstate"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/workspace"
	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/logfiles"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	logsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginprocess"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vaultlist"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func mountCatalog(w *windowWiring) {
	if plugins := getPluginCatalog(w.a); plugins != nil {
		handler := windowcatalog.NewHandler(plugins)
		path, served := windowcatalog.NewRoute(handler, w.state.Gate.CountQuestions())
		w.inst.Mount(editor.NewRoute(path, served, handler))
	}
}

func mountMedia(w *windowWiring) {
	state := w.state
	resolve := file.NewResolveAddress(file.NewStatFile(vaultReaders(w.a.vault)), openPluginFile(w.a.vault), getMediaType(w.a.vault))
	handler := mediawindow.NewHandler(state, state, resolve)
	w.inst.Mount(editor.NewMediaRoute(state, handler))
}

func mountCommand(w *windowWiring) {
	p := w.a
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
	if text := openFileText(w.a.vault); text != nil {
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
		vaultlist.NewOpenVault(registry.Find, w.switching),
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
	finds := newSearch(w.a.search, w.db, w.asking, state.ReportWordsAlone, state.ClearWordsAlone)
	handler := windowsearch.NewHandler(
		state,
		search.NewFindNames(w.db.NoteQueries(), w.kinds.list),
		search.NewFindText(finds, w.db.NoteQueries(), w.db.NoteQueries(), w.kinds.list),
		searchcancel.NewSearches(w.cfg.Clock()),
	)
	setPluginSearch(w.a.search, w.db, w.embedder, state.ReportFloorError, newSearchRuns(w))
	path, served := windowsearch.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func getTabKindChooser(c pluginsWiring) file.TabKindChooser {
	return openChooseTabKind(c)
}

func mountFile(w *windowWiring) {
	state := w.state
	readers := vaultReaders(w.a.vault)
	kinds := w.kinds
	folders := filewindow.NewFolders(
		state,
		file.NewListFolder(file.NewGetFolderEntries(readers, w.db.NoteQueries()), kinds.list),
		file.NewListFileKinds(kinds.files, kinds.chooser),
		file.NewCreateFolder(folderWriters(w.a.vault), state.Writes),
	)
	moveFile := vault.NewMoveFile(&w.cases.Vaults.Tree.Move, state.Writes, state.Writes)
	removeFile := note.NewRemoveFile(&w.cases.Notes.Place.Remove, state.Writes)
	changes := filewindow.NewChanges(state, state.Notices, &moveFile, &removeFile, &w.cases.Notes.Place.EmptyTrash)
	handler := filewindow.NewHandler(folders, changes)
	w.inst.Mount(editor.NewFileRoute(state, handler))
}

func mountWindowState(w *windowWiring) {
	state := w.state
	path, served := windowstate.NewRoute(state.Window, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, state.Window))
}

func mountWorkspace(w *windowWiring) {
	state := w.state
	links := workspace.NewLinks()
	store := w.db.OpenTabs()
	handler := workspace.NewHandler(
		state, state.Notices, state, vault.NewWriteOpenTabs(store), vault.NewReadOpenTabs(store), links,
	)
	path, served := workspace.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
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
		query.NewGetRelatedNotesView(w.db.NoteQueryReads(), w.db.NoteQueries(), w.db.NoteQueries(), w.kinds.list),
		query.NewListHeadings(w.db.NoteQueries()),
	)
	links := notewindow.NewLinks(
		state, query.NewResolveLinks(w.db.NoteQueries()), &notes.Connect.CreateInRelation, &queries.Refer.Backlinks, writes,
	)
	connections := notewindow.NewConnections(
		state, &queries.Browse.Connection, &notes.Connect.SaveConnection, &notes.Connect.RemoveSide, &notes.Connect.RemoveConnection, &notes.Connect.Join,
	)
	handler := notewindow.NewHandler(state.Notices, prose, outlining, links, connections)
	w.inst.Mount(editor.NewNoteRoute(state, handler))
}

func openGetSettings(c Config) *settings.Get {
	return settings.NewGet(readSettings(settingsOf(c)), settingsPlugins(settingsOf(c)), func() []plugin.Setting { return listAppSettings(settingsOf(c)) })
}

func mountSettings(w *windowWiring) {
	state := w.state
	retry := openRetryPlugin(w.a.settings, getPluginRetrier(w.a))
	folders := windowsettings.NewPluginFolders(retry, nil, nil)
	if data := openPluginData(w.a.vault); data != nil {
		folders = windowsettings.NewPluginFolders(retry, data, state)
	}
	values := windowsettings.NewValues(
		openGetSettings(w.cfg),
		openWriteSettings(w.a.settings), openWatchSettings(w.a.settings), nil,
	)
	handler := windowsettings.NewHandler(values, folders)
	w.inst.OnFolderDialog(func(dialog editor.PathDialog) {
		handler.SetPathChooser(openChoosePluginPath(w.a.settings, dialog))
	})
	path, served := windowsettings.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func mountPlugin(w *windowWiring) {
	w.inst.SetPlugins(getPluginServices(w.a), func(id string) (string, bool) { return getPluginDir(w.a, id) })
	host := w.a.PluginHost
	watch := pluginprocess.NewWatchProcesses(host.Processes, host.Health, host.Runtime.PluginResolver, getConnectionResolve(w.a), w.cfg.Clock())
	handler := windowplugin.NewHandler(watch)
	w.inst.Mount(editor.NewRoute(windowplugin.NewRoute(handler, w.state.Gate.CountQuestions())))
}

func mountHealth(w *windowWiring) {
	host := w.a.PluginHost
	handler := windowhealth.NewHandler(
		host.Health, pluginhealth.NewReport(host.Health, host.Runtime.PluginResolver), pluginhealth.NewClear(host.Health),
	)
	path, served := windowhealth.NewRoute(handler, w.state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func openLogEntries(c Config) *logsusecase.Entries {
	if c.Resources != nil && c.Resources.logEntries != nil {
		return c.Resources.logEntries
	}
	return logsusecase.NewEntries()
}

func openRecordLog(c Config) *logsusecase.Record {
	return logsusecase.NewRecord(
		openLogEntries(c),
		logfiles.NewBesideIndex(c.IndexPath),
		logLevelSetting(settingsOf(c)),
		c.Clock(),
	)
}

func openRecordLogFunc(c Config) func(domainlogs.Entry) {
	record := openRecordLog(c)
	return func(entry domainlogs.Entry) {
		if err := record.Execute(entry); err != nil {
			if c.ErrorHandler != nil {
				c.ErrorHandler(err)
			}
		}
	}
}

func mountLogs(w *windowWiring) {
	record := openRecordLog(w.cfg)
	watch := logsusecase.NewWatch(openLogEntries(w.cfg), logfiles.NewBesideIndex(w.cfg.IndexPath), w.cfg.Clock())
	w.state.SetCallTrace(w.cfg.Clock(), openRecordLogFunc(w.cfg))
	w.a.PluginHost.Lender.Update(func(l *hostendpoint.Lent) {
		l.Logs = record
	})
	handler := windowlogs.NewHandler(watch, record, logLevelSetting(settingsOf(w.cfg)), writeLogLevel(settingsOf(w.cfg)))
	path, served := windowlogs.NewRoute(handler, w.state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func shareEditorWindow(db *Index, opened *windowWiring) {
	view := windowView(opened.state)
	focusFile := file.NewFocusFile(vaultReaders(opened.a.vault), view)
	setPluginWindow(opened.a, windowsession.NewPluginWindow(
		&opened.state.Runtime,
		&opened.state.Tabs,
		vaultFocusNote(opened.a.vault, db.NoteQueries(), view),
		windowsession.FileFocusFunc(func(ctx context.Context, v domain.Vault, path string, ranges []domain.ByteRange) error {
			return focusFile.Execute(ctx, v, file.FocusFileInput{Path: path, Ranges: ranges})
		}),
		opened.cases.ShowTab,
	))
}

func shareVaultList(opened *windowWiring) {
	useCases := opened.cases.Vaults
	setPluginVaultList(opened.a, windowsession.NewPluginVaults(&opened.state.Runtime, windowsession.VaultOps{
		Edit:    windowsession.VaultEdits{Add: useCases.Change.Add, Rename: useCases.Change.Rename, Remove: useCases.Change.Remove},
		Find:    opened.cases.Listing.Find,
		Missing: opened.cases.Listing.FolderCheck,
		Open:    opened.cases.OpenPlugin,
	}))
}

func mountVault(w *windowWiring) {
	state := w.state
	handler := windowvault.NewHandler(state, state.Notices)
	path, served := windowvault.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func getReadVault(a pluginsWiring, db *Index) indexing.OpenReading {
	return func(
		ctx context.Context, source searchsourceport.Source, v domain.Vault, rebuild bool,
	) (indexing.ReadWholeVault, error) {
		search := a.search
		search.ShouldRebuildIndex = rebuild
		return readWholeVault(search, ctx, db, source, v, vaultMakeRefresh(a.vault, db, db.Notes()))
	}
}

func newIndexWorker(
	a pluginsWiring, db *Index, state *editor.State, embedding *indexing.Embedding, indexChanged func(context.Context, domain.Vault), out io.Writer,
) *indexing.Worker {
	return indexing.NewWorker(indexing.Config{
		Reading:      getReadVault(a, db),
		Source:       embedding.GetSource(),
		VectorCounts: indexing.NewVectorCounts(embedding, &state.Runtime).Execute,
		IndexChanged: indexChanged,
		Events:       windowsession.NewIndexEvents(&state.Runtime, state.Window, out, a.vault.RecordLog),
		Quiet:        indexing.QuietPeriod,
		After:        a.vault.Clock.After,
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

func getTextVersions(a pluginsWiring) windowsession.TextVersions {
	if versions := a.vault.TextVersions; versions != nil {
		return versions.Changes
	}
	return nil
}

func getTriggerSource(a pluginsWiring, frontmatter *indexnote.FrontmatterChanges) windowsession.TriggerSource {
	return windowsession.TriggerSource{
		Open: func() windowsession.TriggerRunner {
			if triggers := openTriggers(a); triggers != nil {
				return triggers
			}
			return nil
		},
		Watch:            func() (<-chan struct{}, func()) { return watchPluginChanges(a) },
		WatchFrontmatter: frontmatter.Watch,
	}
}

func startVault(a pluginsWiring, db *Index, current *vault.CurrentRefresh) windowsession.VaultPassesFunc {
	return func(
		ctx context.Context, v domain.Vault, rebuild bool, report windowsession.Reporters,
	) windowsession.Started {
		opening := vaultOpener(a.vault, db)
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

func mountConnection(w *windowWiring) {
	handler := windowconnection.NewService(openConnectionUseCases(w.a))
	path, served := windowconnection.NewRoute(handler, w.state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func mountConnectionCall(w *windowWiring) {
	handler := connectioncall.NewService(openModelCalls(w.a), openRowLists(w.a))
	path, served := connectioncall.NewRoute(handler, w.state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func getConnectionResolve(c pluginsWiring) connectionport.ResolveConnection {
	deps, ok := openConnectionDeps(c)
	if !ok {
		return nil
	}
	return connection.NewResolve(deps).Execute
}

func getFeedStart(a pluginsWiring, db *Index) func(ctx context.Context, v domain.Vault) func() {
	events := windowsession.FeedEvents{
		WatchWrites:    db.FeedChanges().Watch,
		WatchPlugins:   func() (<-chan struct{}, func()) { return watchPluginChanges(a) },
		ReadHead:       db.FileChanges().GetHead,
		ListPaths:      db.FileChanges().ListChangedPaths,
		GetContentType: a.PluginHost.Watch.GetContentType,
		Emit: func(v domain.Vault, types []domain.ContentType) {
			a.PluginHost.Runtime.EmitFilesChanged(string(v.ID), types)
		},
		Quiet: windowsession.FeedQuietPeriod,
		After: a.vault.Clock.After,
	}
	return events.Start
}
