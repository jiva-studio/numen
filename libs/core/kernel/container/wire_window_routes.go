package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
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
	windowidentity "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/identity"
	windowkeyfile "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/keyfile"
	windowlicense "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/license"
	windowlogs "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/logs"
	windowplugin "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/plugin"
	windowremotevault "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/remotevault"
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
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfolder"
	machineadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
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
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func mountCatalog(w *windowRoutes, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring) {
	if plugins := getPluginCatalog(install.PluginCatalog, settingsDeps.HandleError); plugins != nil {
		handler := windowcatalog.NewHandler(plugins, exec.Runtime.Listing.ListPlugins)
		w.Installation.Mount(editor.NewRoute(windowcatalog.NewRoute(handler, w.page.Gate.CountQuestions())))
	}
}

func mountMedia(w *windowRoutes, vaultDeps vaultWiring) {
	resolve := file.NewResolveAddress(file.NewStatFile(vaultDeps.Readers), openPluginFile(vaultDeps.Options), getMediaType())
	handler := mediawindow.NewHandler(&w.session.Runtime, &w.page.Media, resolve)
	w.Installation.Mount(editor.NewRoute(mediawindow.NewRoute(handler, w.page.Gate.CountQuestions())))
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
	handler := windowtext.NewUnavailable()
	if versions := vaultDeps.TextVersions; versions != nil {
		handler = windowtext.NewHandler(&w.session.Runtime, textversion.NewFileText(versions.Texts, versions.Versions))
	}
	w.Installation.Mount(editor.NewRoute(windowtext.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func mountVaults(w *windowRoutes, ports registry.Ports, readers vaultport.VaultReaders, change vaultChanging, open *windowsession.OpenVault, dialog editor.PathDialog) {
	handler := vaults.NewHandler(
		&w.session.Runtime,
		vault.NewListVaults(ports.Vaults, readers),
		vault.NewAddVault(change.Add, readers),
		vault.NewRenameVault(ports.Lookup, change.Rename, readers),
		open,
		change.Remove,
	)
	path, served := vaults.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served))
	if dialog != nil {
		path, served = vaults.NewDialogRoute(picker.NewChooseFolder(dialog), picker.NewSaveText(dialog), w.page.Gate.CountQuestions())
		w.Installation.Mount(editor.NewRoute(path, served))
	}
}

func mountSearch(w *windowRoutes, session windowSearch, searchDeps textSearch, lender *hostendpoint.Lender, clock timeSource, runs *searchquery.SearchRuns) {
	status := editor.SearchTasks{Tasks: w.page.Window}
	finds := newSearch(searchDeps, session.index, session.source.searchSource, status.ReportWordsAlone, status.ClearWordsAlone)
	handler := windowsearch.NewHandler(
		&w.session.Runtime,
		search.NewFindNames(session.index.NoteQueries(), session.kinds),
		search.NewFindText(finds, session.index.NoteQueries(), session.index.NoteQueries(), session.kinds),
		searchcancel.NewSearches(clock),
	)
	access := buildPluginSearch(searchDeps, session.index, session.source, status.ReportFloorError, clock).CreateAccess(runs)
	lender.Update(func(l *hostendpoint.Lent) { l.Search = access })
	w.Installation.Mount(editor.NewRoute(windowsearch.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func mountFile(w *windowRoutes, vaultDeps vaultWiring, db *vaultIndex, kinds tabKinds, placing *notePlacing, files file.FileProvider) {
	readers := vaultDeps.Readers
	folders := filewindow.NewFolders(
		&w.session.Runtime,
		file.NewListFolder(file.NewGetFolderEntries(readers, db.NoteQueries()), kinds.list),
		file.NewListFileKinds(kinds.files, kinds.choice),
		file.NewCreateFolder(vaultDeps.Writers, w.page.Writes.AdmitWrite),
	)
	moveFile := note.NewMoveFile(placing.Move, db.Sources(), w.page.Writes.AdmitWrite, w.page.Writes).WithAbsent(vaultDeps.AbsentFiles)
	removeFile := note.NewRemoveFile(&placing.Remove, w.page.Writes)
	changes := filewindow.NewChanges(&w.session.Runtime, w.session.Notices, &moveFile, &removeFile, &placing.EmptyTrash)
	downloads := filewindow.NewDownloads(&w.session.Runtime, file.NewFetchFile(files), file.NewDropDownload(files), file.NewGetDownloadProgress(files))
	handler := filewindow.NewHandler(folders, changes, downloads)
	path, served := filewindow.NewRoute(handler, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served))
}

func mountWindowState(w *windowRoutes) {
	w.Installation.Mount(editor.NewRoute(windowstate.NewRoute(w.page.Window, w.page.Gate.CountQuestions())))
}

func mountWorkspace(w *windowRoutes, db *vaultIndex) {
	links := workspace.NewLinks()
	store := index.NewOpenTabs(db.DB)
	handler := workspace.NewHandler(
		w.session, w.session.Notices, w.session, vault.NewWriteOpenTabs(store), vault.NewReadOpenTabs(store), links,
	)
	w.Installation.Mount(editor.NewRoute(workspace.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func mountNote(w *windowRoutes, notes *noteUseCases, db *vaultIndex, kinds windowport.TabKinds, wake indexing.WriteWake) {
	queries := createQueryUseCases(db)
	writes := w.page.Writes
	prose := notewindow.NewProse(
		&w.session.Runtime, &notes.Edit.Read, &notes.Edit.Write, &notes.Edit.Create, &notes.Place.Rename, &notewindow.Writes{IsIndexOutdated: writes.IsIndexOutdated, WriteWake: wake},
	)
	outlining := notewindow.NewOutline(
		&w.session.Runtime,
		query.NewGetOpeningNote(db.NoteQueries()),
		query.NewGetRelatedNotesView(db.NoteQueryReads(), db.NoteQueries(), db.NoteQueries(), kinds),
		query.NewListHeadings(db.NoteQueries()),
	)
	links := notewindow.NewLinks(
		&w.session.Runtime, query.NewResolveLinks(db.NoteQueries()), &notes.Connect.CreateInRelation, &queries.Refer.Backlinks, writes.IsIndexOutdated,
	)
	connections := notewindow.NewConnections(
		&w.session.Runtime, &queries.Browse.Connection, &notes.Connect.SaveConnection, &notes.Connect.DeleteConnectionSide, &notes.Connect.DeleteConnection, &notes.Connect.Join,
	)
	handler := notewindow.NewHandler(w.session.Notices, prose, outlining, links, connections)
	w.Installation.Mount(editor.NewRoute(notewindow.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func openGetSettings(s settingsWiring) *settings.Get {
	return settings.NewGet(createSettingsStore(s).ReadJSON, s.Infos.GetPluginInfos, func() []plugin.Setting { return listAppSettings(s) }, settingsadapter.GetAppSettingGroups)
}

func mountSettings(w *windowRoutes, settingsDeps settingsWiring, dialog editor.PathDialog) {
	choice := windowsettings.NewUnavailablePathChoice()
	if dialog != nil {
		choice = windowsettings.NewPathChooser(settings.NewChoosePluginPath(dialog, settingsDeps.Infos.GetPluginInfos, getSettingsWriter(settingsDeps)))
	}
	handler := windowsettings.NewHandler(windowsettings.NewValues(openGetSettings(settingsDeps), openWriteSettings(settingsDeps), openWatchSettings(settingsDeps)), choice)
	w.Installation.Mount(editor.NewRoute(windowsettings.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func mountPluginData(w *windowRoutes, vaultDeps vaultWiring) {
	folders := windowsettings.NewUnavailablePluginData()
	if data := openPluginData(vaultDeps.Options, vaultDeps.TextVersions); data != nil {
		folders = windowsettings.NewPluginFolders(data, &w.session.Runtime)
	}
	w.Installation.Mount(editor.NewRoute(windowsettings.NewPluginDataRoute(folders, w.page.Gate.CountQuestions())))
}

func mountPlugin(w *windowRoutes, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	watch := pluginprocess.NewWatchProcesses(exec.Processes, exec.Health, exec.Runtime.Resolver, getConnectionResolve(install, exec, settingsDeps, vaultDeps), vaultDeps.Clock)
	handler := windowplugin.NewHandler(watch, settings.NewRetryPlugin(exec.Runtime.RetryPlugin))
	w.Installation.Mount(editor.NewRoute(windowplugin.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func mountHealth(w *windowRoutes, exec *PluginExecution) {
	handler := windowhealth.NewHandler(
		exec.Health, pluginhealth.NewReport(exec.Health, exec.Runtime.Resolver), pluginhealth.NewClear(exec.Health),
	)
	w.Installation.Mount(editor.NewRoute(windowhealth.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func openRecordLog(indexPath string, entries *logsusecase.Entries, getLevel func() domainlogs.Level, timer timeSource) *logsusecase.Record {
	return logsusecase.NewRecord(entries, logfiles.NewBesideIndex(indexPath), getLevel, timer)
}

func openOptionalRecordLog(indexPath string, res *Resources) func(domainlogs.Entry) {
	if res == nil {
		return nil
	}
	record := openRecordLog(indexPath, getLogEntries(res), res.settingFiles.GetLogLevel, GetClock(res))
	return windowlogs.NewRecordFunc(record, res.ErrorHandler)
}

func mountLogs(w *windowRoutes, entries *logsusecase.Entries, install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	record := openRecordLog(vaultDeps.IndexPath, entries, settingsDeps.SettingsFiles.GetLogLevel, vaultDeps.Clock)
	watch := logsusecase.NewWatch(entries, logfiles.NewBesideIndex(vaultDeps.IndexPath), vaultDeps.Clock)
	w.page.Gate.SetCallTrace(vaultDeps.Clock, vaultDeps.RecordLog)
	install.Endpoints.Lender.Update(func(l *hostendpoint.Lent) {
		l.Logs = record
	})
	handler := windowlogs.NewHandler(watch, record, settingsDeps.SettingsFiles.GetLogLevel, writeLogLevel(settingsDeps))
	w.Installation.Mount(editor.NewRoute(windowlogs.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func shareEditorWindow(db *vaultIndex, opened *windowRoutes, showTab windowsession.ShowTab, install pluginInstall, vaultDeps vaultWiring) {
	view := createWindowView(opened.session.Notices)
	focusFile := file.NewFocusFile(vaultDeps.Readers, view)
	window := hostendpoint.NewWindowAccess(
		&opened.session.Runtime, &opened.session.Tabs, query.NewFocusNote(db.NoteQueries(), view), focusFile, showTab,
	)
	install.Endpoints.Lender.Update(func(l *hostendpoint.Lent) { l.Window = window })
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
	handler := windowvault.NewHandler(w.session, w.session.Notices)
	w.Installation.Mount(editor.NewRoute(windowvault.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func getReadVault(searchDeps vaultIndexPlan, vaultDeps vaultWiring, db *vaultIndex) indexing.OpenReading {
	return func(
		ctx context.Context, source searchsourceport.Source, v domain.Vault, rebuild bool,
	) (indexing.ReadWholeVault, error) {
		search := searchDeps
		search.ShouldRebuildIndex = rebuild
		return readWholeVault(search, ctx, db, source, v, createRefresh(vaultDeps, db, db.Notes()))
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
		CreateTimer:  vaultDeps.Clock.StartTimer,
	})
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
		opening := createVaultOpener(vaultDeps, db)
		opening.ShouldRebuild = rebuild
		var changed func(indexing.VaultChanges)
		if report.NotifyChanged != nil {
			changed = func(m indexing.VaultChanges) { report.NotifyChanged(m.Paths, m.Assets, m.ShouldReload) }
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
	w.Installation.Mount(editor.NewRoute(windowconnection.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func mountConnectionCall(w *windowRoutes, install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	calls := getForwardedCalls(settingsDeps.ConnectionTypes, install.Endpoints.Endpoint)
	handler := connectioncall.NewService(calls, openRowLists(install, exec, settingsDeps, vaultDeps), searchcancel.NewSearches(vaultDeps.Clock))
	w.Installation.Mount(editor.NewRoute(connectioncall.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func mountLicense(w *windowRoutes, install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) {
	handler := windowlicense.NewService(openLicenseUseCases(install, settingsDeps, vaultDeps))
	w.Installation.Mount(editor.NewRoute(windowlicense.NewRoute(handler, w.page.Gate.CountQuestions())))
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
		Quiet:       windowsession.FeedQuietPeriod,
		CreateTimer: vaultDeps.Clock.StartTimer,
	}
	return events.Start
}

func mountTheme(w *windowRoutes, themes *theme.Service) {
	path, served := theme.NewRoute(themes, w.page.Gate.CountQuestions())
	w.Installation.Mount(editor.NewRoute(path, served))
}

func mountIdentity(w *windowRoutes, install pluginInstall, vaultDeps vaultWiring, res *Resources, dialog editor.PathDialog) {
	handler := windowidentity.NewService(openIdentityUseCases(install, vaultDeps, res, dialog))
	w.Installation.Mount(editor.NewRoute(windowidentity.NewRoute(handler, w.page.Gate.CountQuestions())))
}

func mountKeyFile(w *windowRoutes, res *Resources) {
	if res != nil && res.KeyFile != nil {
		w.Installation.Mount(editor.NewRoute(windowkeyfile.NewFileRoute(res.KeyFile, w.page.Gate.CountQuestions())))
	}
}

func mountRemoteVaults(w *windowRoutes, ports registry.Ports, add vault.Add, plugins windowPlugins, now machineport.Clock) {
	folders, user := vaultfolder.Folders{}, machineadapter.UserFolders{}
	w.Installation.Mount(editor.NewRoute(windowremotevault.NewRoute(windowremotevault.NewHandler(
		vault.NewListRemoteVaults(plugins.remoteVaults),
		vault.NewSuggestVaultPath(plugins.newVaultsFolder, user, folders, folders),
		vault.NewOpenRemoteVault(ports.Vaults, ports.Remover, folders, folders, user, add, plugins.remoteVaults, now),
	), w.page.Gate.CountQuestions())))
}
