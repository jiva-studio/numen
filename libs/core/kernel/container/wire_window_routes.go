package container

import (
	"context"

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
	windowlicense "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/license"
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
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
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
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vaultlist"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func mountCatalog(w *OpenedWindow, install pluginInstall, settingsDeps settingsWiring) {
	if plugins := getPluginCatalog(install.PluginCatalog, settingsDeps.ErrorHandler); plugins != nil {
		handler := windowcatalog.NewHandler(plugins)
		path, served := windowcatalog.NewRoute(handler, w.State.Gate.CountQuestions())
		w.Installation.Mount(editor.NewRoute(path, served, handler))
	}
}

func mountMedia(w *OpenedWindow, vaultDeps vaultWiring) {
	state := w.State
	resolve := file.NewResolveAddress(file.NewStatFile(vaultDeps.Readers), openPluginFile(vaultDeps.Options), getMediaType())
	handler := mediawindow.NewHandler(state, state, resolve)
	w.Installation.Mount(editor.NewMediaRoute(state, handler))
}

func mountCommand(w *OpenedWindow, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	var versions plugincommand.VersionItems
	if menu := openVersionItems(install, exec, settingsDeps, vaultDeps); menu != nil {
		versions = menu
	}
	commands := newPluginCommands(getWhenConfig(settingsDeps, exec.Runtime.PluginLister), vaultDeps, exec, versions)
	handler := command.NewHandler(w.State, commands.Listing, commands.RunCommand, commands.ListItems)
	w.Installation.Mount(editor.NewRoute(command.NewRoute(w.State.Gate.CountQuestions(), handler)))
}

func mountText(w *OpenedWindow, vaultDeps vaultWiring) {
	handler := windowtext.NewHandler(w.State, nil)
	if text := openFileText(vaultDeps.TextVersions); text != nil {
		handler = windowtext.NewHandler(w.State, text)
	}
	path, served := windowtext.NewRoute(handler, w.State.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountVaults(w *OpenedWindow, listing *registry.Listing, change VaultChanging, switching windowsession.UserSwitch) {
	state := w.State
	handler := vaults.NewHandler(
		state,
		vaultlist.NewListVaults(listing.List, listing.FolderCheck),
		vaultlist.NewAddVault(change.Add, listing.FolderCheck),
		vaultlist.NewRenameVault(listing.Find, change.Rename, listing.FolderCheck),
		vaultlist.NewOpenVault(listing.Find, switching),
		change.Remove,
	)
	w.Installation.OnFolderDialog(func(dialog editor.PathDialog) {
		handler.SetFolderChooser(picker.NewChooseFolder(dialog))
	})
	path, served := vaults.NewRoute(handler, state.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountSearch(w *OpenedWindow, session windowSearch, searchDeps textSearch, lender *hostendpoint.Lender, clock machineport.Timer, runs *searchrun.SearchRuns) {
	state := w.State
	finds := newSearch(searchDeps, session.index, session.source, state.ReportWordsAlone, state.ClearWordsAlone)
	handler := windowsearch.NewHandler(
		state,
		search.NewFindNames(session.index.NoteQueries(), session.kinds),
		search.NewFindText(finds, session.index.NoteQueries(), session.index.NoteQueries(), session.kinds),
		searchcancel.NewSearches(clock),
	)
	access := buildPluginSearch(searchDeps, session.index, session.source, state.ReportFloorError, clock).Access(runs)
	lender.Update(func(l *hostendpoint.Lent) { l.Search = access })
	path, served := windowsearch.NewRoute(handler, state.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountFile(w *OpenedWindow, vaultDeps vaultWiring, db *Index, kinds tabKinds, moving *vault.Move, placing *NotePlacing) {
	state := w.State
	readers := vaultDeps.Readers
	folders := filewindow.NewFolders(
		state,
		file.NewListFolder(file.NewGetFolderEntries(readers, db.NoteQueries()), kinds.list),
		file.NewListFileKinds(kinds.files, kinds.chooser),
		file.NewCreateFolder(vaultDeps.Writers, state.Writes),
	)
	moveFile := vault.NewMoveFile(moving, state.Writes, state.Writes)
	removeFile := note.NewRemoveFile(&placing.Remove, state.Writes)
	changes := filewindow.NewChanges(state, state.Notices, &moveFile, &removeFile, &placing.EmptyTrash)
	handler := filewindow.NewHandler(folders, changes)
	w.Installation.Mount(editor.NewFileRoute(state, handler))
}

func mountWindowState(w *OpenedWindow) {
	state := w.State
	path, served := windowstate.NewRoute(state.Window, state.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, state.Window))
}

func mountWorkspace(w *OpenedWindow, db *Index) {
	state := w.State
	links := workspace.NewLinks()
	store := db.OpenTabs()
	handler := workspace.NewHandler(
		state, state.Notices, state, vault.NewWriteOpenTabs(store), vault.NewReadOpenTabs(store), links,
	)
	path, served := workspace.NewRoute(handler, state.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountNote(w *OpenedWindow, notes *NoteUseCases, db *Index, kinds *file.ListTabKinds) {
	state := w.State
	queries := vaultQueries(db)
	writes := state.Writes
	prose := notewindow.NewProse(
		state, &notes.Edit.Read, &notes.Edit.Write, &notes.Edit.Create, &notes.Place.Rename, writes,
	)
	outlining := notewindow.NewOutline(
		state,
		query.NewGetOpeningNote(db.NoteQueries()),
		query.NewGetRelatedNotesView(db.NoteQueryReads(), db.NoteQueries(), db.NoteQueries(), kinds),
		query.NewListHeadings(db.NoteQueries()),
	)
	links := notewindow.NewLinks(
		state, query.NewResolveLinks(db.NoteQueries()), &notes.Connect.CreateInRelation, &queries.Refer.Backlinks, writes,
	)
	connections := notewindow.NewConnections(
		state, &queries.Browse.Connection, &notes.Connect.SaveConnection, &notes.Connect.RemoveSide, &notes.Connect.RemoveConnection, &notes.Connect.Join,
	)
	handler := notewindow.NewHandler(state.Notices, prose, outlining, links, connections)
	w.Installation.Mount(editor.NewNoteRoute(state, handler))
}

func openGetSettings(s settingsWiring) *settings.Get {
	return settings.NewGet(readSettings(s), s.Infos.GetPluginInfos, func() []plugin.Setting { return listAppSettings(s) })
}

func mountSettings(w *OpenedWindow, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	state := w.State
	retry := settings.NewRetryPlugin(exec.Runtime.RetryPlugin)
	folders := windowsettings.NewPluginFolders(retry, nil, nil)
	if data := openPluginData(vaultDeps.Options, vaultDeps.TextVersions); data != nil {
		folders = windowsettings.NewPluginFolders(retry, data, state)
	}
	values := windowsettings.NewValues(
		openGetSettings(settingsDeps),
		openWriteSettings(settingsDeps), openWatchSettings(settingsDeps), nil,
	)
	handler := windowsettings.NewHandler(values, folders)
	w.Installation.OnFolderDialog(func(dialog editor.PathDialog) {
		handler.SetPathChooser(openChoosePluginPath(settingsDeps, dialog))
	})
	path, served := windowsettings.NewRoute(handler, state.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountPlugin(w *OpenedWindow, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	w.Installation.SetPlugins(GetPluginServices(exec.Runtime), func(id string) (string, bool) { return GetPluginDir(exec.Runtime.PluginResolver, id) })
	host := exec
	watch := pluginprocess.NewWatchProcesses(host.Processes, host.Health, host.Runtime.PluginResolver, getConnectionResolve(install, exec, settingsDeps, vaultDeps), vaultDeps.Clock)
	handler := windowplugin.NewHandler(watch)
	w.Installation.Mount(editor.NewRoute(windowplugin.NewRoute(handler, w.State.Gate.CountQuestions())))
}

func mountHealth(w *OpenedWindow, exec *PluginExecution) {
	host := exec
	handler := windowhealth.NewHandler(
		host.Health, pluginhealth.NewReport(host.Health, host.Runtime.PluginResolver), pluginhealth.NewClear(host.Health),
	)
	path, served := windowhealth.NewRoute(handler, w.State.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func openRecordLog(indexPath string, entries *logsusecase.Entries, getLevel func() domainlogs.Level, timer machineport.Timer) *logsusecase.Record {
	return logsusecase.NewRecord(entries, logfiles.NewBesideIndex(indexPath), getLevel, timer)
}

func recordLogFunc(record *logsusecase.Record, report func(error)) func(domainlogs.Entry) {
	return func(entry domainlogs.Entry) {
		if err := record.Execute(entry); err != nil && report != nil {
			report(err)
		}
	}
}

func openOptionalRecordLog(indexPath string, res *Resources) func(domainlogs.Entry) {
	if res == nil {
		return nil
	}
	record := openRecordLog(indexPath, getLogEntries(res), res.settingsFiles.GetLogLevel, GetClock(res))
	return recordLogFunc(record, res.ErrorHandler)
}

func mountLogs(w *OpenedWindow, entries *logsusecase.Entries, install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	record := openRecordLog(vaultDeps.IndexPath, entries, settingsDeps.SettingsFiles.GetLogLevel, vaultDeps.Clock)
	watch := logsusecase.NewWatch(entries, logfiles.NewBesideIndex(vaultDeps.IndexPath), vaultDeps.Clock)
	w.State.SetCallTrace(vaultDeps.Clock, vaultDeps.RecordLog)
	install.Endpoints.Lender.Update(func(l *hostendpoint.Lent) {
		l.Logs = record
	})
	handler := windowlogs.NewHandler(watch, record, settingsDeps.SettingsFiles.GetLogLevel, writeLogLevel(settingsDeps))
	path, served := windowlogs.NewRoute(handler, w.State.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func shareEditorWindow(db *Index, opened *OpenedWindow, showTab windowsession.ShowTab, install pluginInstall, vaultDeps vaultWiring) {
	view := windowView(opened.State)
	focusFile := file.NewFocusFile(vaultDeps.Readers, view)
	setPluginWindow(install.Endpoints.Lender, windowsession.NewPluginWindow(
		&opened.State.Runtime,
		&opened.State.Tabs,
		query.NewFocusNote(db.NoteQueries(), view),
		windowsession.FileFocusFunc(func(ctx context.Context, v domain.Vault, path string, ranges []domain.ByteRange) error {
			return focusFile.Execute(ctx, v, file.FocusFileInput{Path: path, Ranges: ranges})
		}),
		showTab,
	))
}

func newPluginVaultList(state *editor.State, changes VaultChanging, listing registry.Listing, open windowsession.OpenPluginVault) windowsession.PluginVaults {
	return windowsession.NewPluginVaults(&state.Runtime, windowsession.VaultOps{
		Edit:    windowsession.VaultEdits{Add: changes.Add, Rename: changes.Rename, Remove: changes.Remove},
		Find:    listing.Find,
		Missing: listing.FolderCheck,
		Open:    open,
	})
}

func mountVault(w *OpenedWindow) {
	state := w.State
	handler := windowvault.NewHandler(state, state.Notices)
	path, served := windowvault.NewRoute(handler, state.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func getReadVault(searchDeps vaultIndexPlan, vaultDeps vaultWiring, db *Index) indexing.OpenReading {
	return func(
		ctx context.Context, source searchsourceport.Source, v domain.Vault, rebuild bool,
	) (indexing.ReadWholeVault, error) {
		search := searchDeps
		search.ShouldRebuildIndex = rebuild
		return readWholeVault(search, ctx, db, source, v, vaultMakeRefresh(vaultDeps, db, db.Notes()))
	}
}

func newIndexWorker(
	s windowStartup, searchDeps vaultIndexPlan, vaultDeps vaultWiring, state *editor.State, embedding *indexing.Embedding, indexChanged func(context.Context, domain.Vault),
) *indexing.Worker {
	return indexing.NewWorker(indexing.Config{
		Reading:      getReadVault(searchDeps, vaultDeps, s.db),
		Source:       embedding.GetSource(),
		VectorCounts: indexing.NewVectorCounts(embedding, &state.Runtime).Execute,
		IndexChanged: indexChanged,
		Events:       windowsession.NewIndexEvents(&state.Runtime, state.Window, s.opening.Out, vaultDeps.RecordLog),
		Quiet:        indexing.QuietPeriod,
		After:        vaultDeps.Clock.After,
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

func getTextVersions(versions *TextVersions) windowsession.TextVersions {
	if versions != nil {
		return versions.Changes
	}
	return nil
}

func getTriggerSource(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring, frontmatter *indexnote.FrontmatterChanges) windowsession.TriggerSource {
	return windowsession.TriggerSource{
		Open: func() windowsession.TriggerRunner {
			if triggers := openTriggers(install, exec, settingsDeps, vaultDeps); triggers != nil {
				return triggers
			}
			return nil
		},
		Watch:            settingsDeps.PluginChanges.Subscribe,
		WatchFrontmatter: frontmatter.Watch,
	}
}

func startVault(vaultDeps vaultWiring, db *Index, current *vault.CurrentRefresh) windowsession.VaultPassesFunc {
	return func(
		ctx context.Context, v domain.Vault, rebuild bool, report windowsession.Reporters,
	) windowsession.Started {
		opening := vaultOpener(vaultDeps, db)
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

func mountConnection(w *OpenedWindow, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	handler := windowconnection.NewService(openConnectionUseCases(install, exec, settingsDeps, vaultDeps))
	path, served := windowconnection.NewRoute(handler, w.State.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountConnectionCall(w *OpenedWindow, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	handler := connectioncall.NewService(openModelCalls(install, settingsDeps), openRowLists(install, exec, settingsDeps, vaultDeps))
	path, served := connectioncall.NewRoute(handler, w.State.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountLicense(w *OpenedWindow, install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	handler := windowlicense.NewService(openLicenseUseCases(install, settingsDeps, vaultDeps))
	path, served := windowlicense.NewRoute(handler, w.State.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func getConnectionResolve(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) connectionport.ResolveConnection {
	deps, ok := openConnectionDeps(install, exec, settingsDeps, vaultDeps)
	if !ok {
		return nil
	}
	return connection.NewResolve(deps).Execute
}

func getFeedStart(exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring, db *Index) func(ctx context.Context, v domain.Vault) func() {
	events := windowsession.FeedEvents{
		WatchWrites:    db.FeedChanges().Watch,
		WatchPlugins:   settingsDeps.PluginChanges.Subscribe,
		ReadHead:       db.FileChanges().GetHead,
		ListFiles:      db.FileChanges().ListChangedFiles,
		GetContentType: exec.Watch.GetContentType,
		Emit: func(v domain.Vault, types []domain.FileType) {
			exec.Runtime.EmitFilesChanged(string(v.ID), types)
		},
		Quiet: windowsession.FeedQuietPeriod,
		After: vaultDeps.Clock.After,
	}
	return events.Start
}
