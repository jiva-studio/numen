package container

import (
	"context"

	indexnote "github.com/jiva-studio/numen/libs/core/kernel/adapter/index/note"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	pluginvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
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
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/theme"
	windowvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vaults"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/windowstate"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/workspace"
	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/logfiles"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
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
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

func mountCatalog(w *windowRoutes, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring) {
	if plugins := getPluginCatalog(install.PluginCatalog, settingsDeps.ErrorHandler); plugins != nil {
		handler := windowcatalog.NewHandler(plugins, exec.Runtime.Listing.ListPlugins)
		path, served := windowcatalog.NewRoute(handler, w.page.Gate.CountQuestions())
		w.Installation.Mount(editor.NewRoute(path, served, handler))
	}
}

func mountMedia(w *windowRoutes, vaultDeps vaultWiring) {
	state := w.session
	resolve := file.NewResolveAddress(file.NewStatFile(vaultDeps.Readers), openPluginFile(vaultDeps.Options), getMediaType())
	handler := mediawindow.NewHandler(&state.Runtime, &w.page.Media, resolve)
	path, served := mediawindow.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountCommand(w *windowRoutes, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	var versions plugincommand.VersionItems
	if menu := openVersionItems(install, exec, settingsDeps, vaultDeps); menu != nil {
		versions = menu
	}
	commands := newPluginCommands(getWhenConfig(settingsDeps, exec.Runtime.Listing), vaultDeps, exec, versions)
	handler := command.NewHandler(&w.session.Runtime, commands.Listing, commands.RunCommand, commands.ListItems)
	w.Installation.Mount(editor.NewRoute(command.NewRoute(w.page.Gate.CountQuestions(), handler)))
	w.Installation.Mount(editor.NewRoute(command.NewContributionRoute(w.page.Gate.CountQuestions(), handler)))
	w.Installation.Mount(editor.NewRoute(command.NewVersionRoute(w.page.Gate.CountQuestions(), handler)))
}

func mountText(w *windowRoutes, vaultDeps vaultWiring) {
	var handler numenv1connect.TextServiceHandler = numenv1connect.UnimplementedTextServiceHandler{}
	if text := openFileText(vaultDeps.TextVersions); text != nil {
		handler = windowtext.NewHandler(&w.session.Runtime, text)
	}
	path, served := windowtext.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountVaults(w *windowRoutes, ports registry.Ports, readers vaultport.VaultReaders, change vaultChanging, open *windowsession.OpenVault, dialog editor.PathDialog) {
	state := w.session
	handler := vaults.NewHandler(
		&state.Runtime,
		vault.NewListVaults(ports.Vaults, readers),
		vault.NewAddVault(change.Add, readers),
		vault.NewRenameVault(ports.Lookup, change.Rename, readers),
		open,
		change.Remove,
	)
	if dialog != nil {
		handler = handler.WithFolderChooser(picker.NewChooseFolder(dialog))
	}
	path, served := vaults.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
	path, served = vaults.NewDialogRoute(handler.Dialog, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler.Dialog))
}

func mountSearch(w *windowRoutes, session windowSearch, searchDeps textSearch, lender *hostendpoint.Lender, clock installationClock, runs *searchquery.SearchRuns) {
	state := w.session
	status := editor.SearchTasks{Tasks: w.page.Window}
	finds := newSearch(searchDeps, session.index, session.source.searchSource, status.ReportWordsAlone, status.ClearWordsAlone)
	handler := windowsearch.NewHandler(
		&state.Runtime,
		search.NewFindNames(session.index.NoteQueries(), session.kinds),
		search.NewFindText(finds, session.index.NoteQueries(), session.index.NoteQueries(), session.kinds),
		searchcancel.NewSearches(clock),
	)
	access := buildPluginSearch(searchDeps, session.index, session.source, status.ReportFloorError, clock).Access(runs)
	lender.Update(func(l *hostendpoint.Lent) { l.Search = access })
	path, served := windowsearch.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountFile(w *windowRoutes, vaultDeps vaultWiring, db *vaultIndex, kinds tabKinds, placing *notePlacing) {
	state := w.session
	readers := vaultDeps.Readers
	folders := filewindow.NewFolders(
		&state.Runtime,
		file.NewListFolder(file.NewGetFolderEntries(readers, db.NoteQueries()), kinds.list),
		file.NewListFileKinds(kinds.files, kinds.choice),
		file.NewCreateFolder(vaultDeps.Writers, w.page.Writes.AdmitWrite),
	)
	moveFile := note.NewMoveFile(placing.Move, db.Sources(), w.page.Writes.AdmitWrite, w.page.Writes)
	removeFile := note.NewRemoveFile(&placing.Remove, w.page.Writes)
	changes := filewindow.NewChanges(&state.Runtime, state.Notices, &moveFile, &removeFile, &placing.EmptyTrash)
	handler := filewindow.NewHandler(folders, changes)
	path, served := filewindow.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountWindowState(w *windowRoutes) {
	path, served := windowstate.NewRoute(w.page.Window, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, w.page.Window))
}

func mountWorkspace(w *windowRoutes, db *vaultIndex) {
	state := w.session
	links := workspace.NewLinks()
	store := db.OpenTabs()
	handler := workspace.NewHandler(
		state, state.Notices, state, vault.NewWriteOpenTabs(store), vault.NewReadOpenTabs(store), links,
	)
	path, served := workspace.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountNote(w *windowRoutes, notes *noteUseCases, db *vaultIndex, kinds windowport.TabKinds, wake indexing.WriteWake) {
	state := w.session
	queries := vaultQueries(db)
	writes := w.page.Writes
	prose := notewindow.NewProse(
		&state.Runtime, &notes.Edit.Read, &notes.Edit.Write, &notes.Edit.Create, &notes.Place.Rename, &notewindow.Writes{IsIndexOutdated: writes.IsIndexOutdated, WriteWake: wake},
	)
	outlining := notewindow.NewOutline(
		&state.Runtime,
		query.NewGetOpeningNote(db.NoteQueries()),
		query.NewGetRelatedNotesView(db.NoteQueryReads(), db.NoteQueries(), db.NoteQueries(), kinds),
		query.NewListHeadings(db.NoteQueries()),
	)
	links := notewindow.NewLinks(
		&state.Runtime, query.NewResolveLinks(db.NoteQueries()), &notes.Connect.CreateInRelation, &queries.Refer.Backlinks, writes.IsIndexOutdated,
	)
	connections := notewindow.NewConnections(
		&state.Runtime, &queries.Browse.Connection, &notes.Connect.SaveConnection, &notes.Connect.DeleteConnectionSide, &notes.Connect.DeleteConnection, &notes.Connect.Join,
	)
	handler := notewindow.NewHandler(state.Notices, prose, outlining, links, connections)
	path, served := notewindow.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func openGetSettings(s settingsWiring) *settings.Get {
	return settings.NewGet(readSettings(s), s.Infos.GetPluginInfos, func() []plugin.Setting { return listAppSettings(s) }, settingsadapter.GetAppSettingGroups)
}

func mountSettings(w *windowRoutes, settingsDeps settingsWiring, dialog editor.PathDialog) {
	var choice windowsettings.PathChoice = numenv1connect.UnimplementedSettingsServiceHandler{}
	if dialog != nil {
		choice = windowsettings.NewPathChooser(openChoosePluginPath(settingsDeps, dialog))
	}
	handler := windowsettings.NewHandler(windowsettings.NewValues(openGetSettings(settingsDeps), openWriteSettings(settingsDeps), openWatchSettings(settingsDeps)), choice)
	path, served := windowsettings.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountPluginData(w *windowRoutes, vaultDeps vaultWiring) {
	var folders numenv1connect.PluginDataServiceHandler = numenv1connect.UnimplementedPluginDataServiceHandler{}
	if data := openPluginData(vaultDeps.Options, vaultDeps.TextVersions); data != nil {
		folders = windowsettings.NewPluginFolders(data, &w.session.Runtime)
	}
	path, served := windowsettings.NewPluginDataRoute(folders, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, folders))
}

func mountPlugin(w *windowRoutes, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	host := exec
	watch := pluginprocess.NewWatchProcesses(host.Processes, host.Health, host.Runtime.Resolver, getConnectionResolve(install, exec, settingsDeps, vaultDeps), vaultDeps.Clock)
	handler := windowplugin.NewHandler(watch, settings.NewRetryPlugin(exec.Runtime.RetryPlugin))
	w.Installation.Mount(editor.NewRoute(windowplugin.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func mountHealth(w *windowRoutes, exec *PluginExecution) {
	host := exec
	handler := windowhealth.NewHandler(
		host.Health, pluginhealth.NewReport(host.Health, host.Runtime.Resolver), pluginhealth.NewClear(host.Health),
	)
	path, served := windowhealth.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func openRecordLog(indexPath string, entries *logsusecase.Entries, getLevel func() domainlogs.Level, timer installationClock) *logsusecase.Record {
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

func mountLogs(w *windowRoutes, entries *logsusecase.Entries, install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	record := openRecordLog(vaultDeps.IndexPath, entries, settingsDeps.SettingsFiles.GetLogLevel, vaultDeps.Clock)
	watch := logsusecase.NewWatch(entries, logfiles.NewBesideIndex(vaultDeps.IndexPath), vaultDeps.Clock)
	w.page.Gate.SetCallTrace(vaultDeps.Clock, vaultDeps.RecordLog)
	install.Endpoints.Lender.Update(func(l *hostendpoint.Lent) {
		l.Logs = record
	})
	handler := windowlogs.NewHandler(watch, record, settingsDeps.SettingsFiles.GetLogLevel, writeLogLevel(settingsDeps))
	path, served := windowlogs.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func shareEditorWindow(db *vaultIndex, opened *windowRoutes, showTab windowsession.ShowTab, install pluginInstall, vaultDeps vaultWiring) {
	view := windowView(opened.session.Notices)
	focusFile := file.NewFocusFile(vaultDeps.Readers, view)
	setPluginWindow(install.Endpoints.Lender, hostendpoint.NewWindowAccess(
		&opened.session.Runtime, &opened.session.Tabs, query.NewFocusNote(db.NoteQueries(), view), focusFile, showTab,
	))
}

func newPluginVaultList(
	runtime *windowsession.Runtime, changes vaultChanging, lookup vaultport.VaultLookup, readers vaultport.VaultReaders, open windowsession.OpenPluginVault,
) pluginvault.ListAccess {
	return pluginvault.NewListAccess(
		vault.NewAddVault(changes.Add, readers), vault.NewRenameVault(lookup, changes.Rename, readers), changes.Remove, open,
		registry.NewFolderCheck(readers), func() domain.VaultID { return runtime.GetCurrentVault().ID },
	)
}

func mountVault(w *windowRoutes) {
	state := w.session
	handler := windowvault.NewHandler(state, state.Notices)
	path, served := windowvault.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func getReadVault(searchDeps vaultIndexPlan, vaultDeps vaultWiring, db *vaultIndex) indexing.OpenReading {
	return func(
		ctx context.Context, source searchsourceport.Source, v domain.Vault, rebuild bool,
	) (indexing.ReadWholeVault, error) {
		search := searchDeps
		search.ShouldRebuildIndex = rebuild
		return readWholeVault(search, ctx, db, source, v, vaultMakeRefresh(vaultDeps, db, db.Notes()))
	}
}

func (s windowStartup) newIndexWorker(
	searchDeps vaultIndexPlan, vaultDeps vaultWiring, runtime *windowsession.Runtime, tasks windowport.Tasks, embedding *indexing.Embedding, indexChanged func(context.Context, domain.Vault),
) *indexing.Worker {
	return indexing.NewWorker(indexing.Config{
		Reading:      getReadVault(searchDeps, vaultDeps, s.db),
		Source:       embedding.GetSource(),
		VectorCounts: indexing.NewVectorCounts(embedding, runtime).Execute,
		IndexChanged: indexChanged,
		Events:       windowsession.NewIndexEvents(runtime, tasks, s.opening.Out, vaultDeps.RecordLog),
		Quiet:        indexing.QuietPeriod,
		After:        vaultDeps.Clock.After,
	})
}

func newIndexSession(worker *indexing.Worker) func(domain.Vault) windowsession.IndexSession {
	return func(v domain.Vault) windowsession.IndexSession {
		session := worker.NewSession(v)
		return windowsession.IndexSession{
			Run: func(ctx context.Context, rebuild bool, read windowsession.ReadNotes) {
				session.Run(ctx, rebuild, read)
			},
			ReportVersionsChanged: session.ReportVersionsChanged,
		}
	}
}

func getTextVersions(versions *textVersions) windowsession.TextVersions {
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

func startVault(vaultDeps vaultWiring, db *vaultIndex, current *indexing.CurrentRefresh) windowsession.VaultPassesFunc {
	return func(
		ctx context.Context, v domain.Vault, rebuild bool, report windowsession.Reporters,
	) windowsession.Started {
		opening := vaultOpener(vaultDeps, db)
		opening.ShouldRebuild = rebuild
		var changed func(indexing.VaultChanges)
		if report.Changed != nil {
			changed = func(m indexing.VaultChanges) { report.Changed(m.Paths, m.Assets, m.ShouldReload) }
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

func mountConnection(w *windowRoutes, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	handler := windowconnection.NewService(openConnectionUseCases(install, exec, settingsDeps, vaultDeps))
	path, served := windowconnection.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountConnectionCall(w *windowRoutes, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring, clock installationClock) {
	handler := connectioncall.NewService(openModelCalls(install, settingsDeps), openRowLists(install, exec, settingsDeps, vaultDeps), searchcancel.NewSearches(clock))
	path, served := connectioncall.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func mountLicense(w *windowRoutes, install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	handler := windowlicense.NewService(openLicenseUseCases(install, settingsDeps, vaultDeps))
	path, served := windowlicense.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, handler))
}

func getConnectionResolve(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) connectionport.ResolveConnection {
	deps, ok := openConnectionDeps(install, exec, settingsDeps, vaultDeps)
	if !ok {
		return nil
	}
	return connection.NewResolve(deps).Execute
}

func getFeedStart(exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring, db *vaultIndex) func(ctx context.Context, v domain.Vault) func() {
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

func mountTheme(w *windowRoutes, themes *theme.Service) {
	path, served := theme.NewRoute(themes, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served, themes))
}
