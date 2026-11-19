// Package container is the composition root: the only place that selects which
// concrete adapter satisfies which port.
//
// Use cases above it work with the interfaces in core/port. To replace the index or
// the vault reader, only this package changes.
package container

import (
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	windowlogs "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/logs"
	machinedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	logsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
)

// Paths are the files and folders the user may point somewhere else. An empty path means the platform's
// own location.
type Paths struct {
	IndexPath       string
	RegistryPath    string
	ServiceDir      string
	SettingsPath    string
	ThemesPath      string
	PluginCachePath string
}

// Scales are the sizes of the interface and of the text; zero means the settings decide.
type Scales struct {
	InterfaceScale, TextScale float64
}

// Machine is what the hardware allows the installation to use.
type Machine struct {
	ResolvedCompute machinedomain.ResolvedCompute
}

// Config holds the launch settings and the resolved machine plan.
type Config struct {
	Paths
	ShouldRebuildIndex bool
	SafeMode           bool
	machine            Machine
}

// Resources holds the services shared by the parts of one installation.
type Resources struct {
	PluginCatalog catalog.Config
	KeyStore      pluginport.KeyStore
	ErrorHandler  window.ErrorHandler
	Now           machineport.Timer
	Chunks        *note.ChunkTokens
	TextReaders   textdomain.Readers

	vaultLocks      *vaultwriter.Locks
	watches         *vaultwatch.Hub
	settingsFiles   *settingsadapter.Files
	settingEvents   *settingevents.Events
	pluginChanges   *settingsusecase.PluginChanges
	subscribedHosts settingsusecase.HostSubscriptions
	logEntries      *logsusecase.Entries
}

func NewResources() *Resources {
	return &Resources{
		Chunks:        &note.ChunkTokens{},
		vaultLocks:    vaultwriter.NewLocks(),
		watches:       vaultwatch.NewHub(),
		settingsFiles: settingsadapter.NewFiles(),
		settingEvents: settingevents.New(),
		pluginChanges: &settingsusecase.PluginChanges{},
		logEntries:    logsusecase.NewEntries(),
	}
}

func getLogEntries(res *Resources) *logsusecase.Entries {
	if res.logEntries != nil {
		return res.logEntries
	}
	return logsusecase.NewEntries()
}

// PreparePluginHost selects the installation's roles and applies its launch mode.
func PreparePluginHost(exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, safeMode bool) (*PluginExecution, *PluginPreferences, *PluginEndpoints) {
	exec = getPluginExecution(exec)
	exec.Runtime.SetSafeMode(safeMode)
	return exec, getPluginPreferences(prefs), getPluginEndpoints(endpoints)
}

// GetMachine returns the machine plan resolved from the settings.
func (c Config) GetMachine() Machine { return c.machine }

func getPluginExecution(exec *PluginExecution) *PluginExecution {
	if exec == nil {
		return newAbsentExecution(plugindomain.NotInstalled, nil)
	}
	return exec
}

func getPluginPreferences(prefs *PluginPreferences) *PluginPreferences {
	if prefs == nil {
		return newAbsentPreferences()
	}
	return prefs
}

func getPluginEndpoints(endpoints *PluginEndpoints) *PluginEndpoints {
	if endpoints == nil {
		return newAbsentEndpoints()
	}
	return endpoints
}

func getRunningExecution(exec *PluginExecution, safeMode bool) *PluginExecution {
	if safeMode {
		return newAbsentExecution(plugindomain.Disabled, errSafeMode)
	}
	return getPluginExecution(exec)
}

// GetClock returns the clock every use case assembled here is handed.
func GetClock(res *Resources) machineport.Timer {
	if res.Now != nil {
		return res.Now
	}
	return clock.System{}
}

func getErrorHandler(indexPath string, res *Resources) window.ErrorHandler {
	return windowlogs.ReportError(openOptionalRecordLog(indexPath, res), res.ErrorHandler, "Background operation failed")
}

func settingsOf(c Config, exec *PluginExecution, prefs *PluginPreferences, res *Resources) settingsWiring {
	runtime := getPluginExecution(exec).Runtime
	prefs = getPluginPreferences(prefs)
	return settingsWiring{
		settingsPlaces: settingsPlaces{
			SettingsPath:    c.SettingsPath,
			ThemesPath:      c.ThemesPath,
			RegistryPath:    c.RegistryPath,
			ResolvedCompute: c.machine.ResolvedCompute,
		},
		settingsStores: settingsStores{
			Watches:         res.watches,
			SettingsFiles:   res.settingsFiles,
			SettingEvents:   res.settingEvents,
			PluginChanges:   res.pluginChanges,
			SubscribedHosts: &res.subscribedHosts,
		},
		Infos:         runtime,
		Configuration: prefs.Configuration,
		Views:         prefs.Views,
		ApplySetting:  prefs.ApplySetting,
		Detection:     prefs.Detection,
		ErrorHandler:  getErrorHandler(c.IndexPath, res),
	}
}

func searchWith(c Config, exec, running *PluginExecution, endpoints *PluginEndpoints, res *Resources, v vaultWiring) searchWiring {
	return searchWiring{
		ServiceDir:         c.ServiceDir,
		ShouldRebuildIndex: c.ShouldRebuildIndex,
		Chunks:             res.Chunks,
		ResolvedCompute:    c.machine.ResolvedCompute,
		Runtime:            exec.Runtime,
		Watch:              exec.Watch,
		Health:             exec.Health,
		ActiveRuntime:      running.Runtime,
		Sources:            endpoints.SourceDescriptions,
		Lender:             endpoints.Lender,
		Clock:              GetClock(res),
		VaultReaders:       vaultReaders(v),
		TextReaders:        getTextReaders(v),
		ErrorHandler:       getErrorHandler(c.IndexPath, res),
	}
}

func vaultBase(c Config, running *PluginExecution, endpoints *PluginEndpoints, res *Resources, texts *TextVersions) vaultWiring {
	return vaultWiring{
		IndexPath:    c.IndexPath,
		RegistryPath: c.RegistryPath,
		ServiceDir:   c.ServiceDir,
		Clock:        GetClock(res),
		Lender:       endpoints.Lender,
		TextPlugins:  running.Runtime.TextPlugins,
		TextVersions: texts,
		TextReaders:  res.TextReaders,
		VaultLocks:   res.vaultLocks,
		Watches:      res.watches,
		ErrorHandler: getErrorHandler(c.IndexPath, res),
	}
}

func vaultWith(res *Resources, base vaultWiring, settings settingsWiring, search searchWiring) vaultWiring {
	base.SyncSetting = syncSetting(settings)
	base = setVaultSearch(base, search)
	record := openRecordLog(base.IndexPath, getLogEntries(res), logLevelSetting(settings), base.Clock)
	base.RecordLog = recordLogFunc(record, res.ErrorHandler)
	return base
}

func assemble(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, texts *TextVersions) (pluginInstall, settingsWiring, searchWiring, vaultWiring) {
	// texts are the text versions the open made over its index, or nil where it makes none.
	exec = getPluginExecution(exec)
	running := getRunningExecution(exec, c.SafeMode)
	endpoints = getPluginEndpoints(endpoints)
	settings := settingsOf(c, exec, prefs, res)
	base := vaultBase(c, running, endpoints, res, texts)
	search := searchWith(c, exec, running, endpoints, res, base)
	install := pluginInstall{
		PluginCachePath: c.PluginCachePath,
		ServiceDir:      c.ServiceDir,
		Execution:       exec,
		ActiveExecution: running,
		Endpoints:       endpoints,
		PluginCatalog:   res.PluginCatalog,
		KeyStore:        res.KeyStore,
	}
	return install, settings, search, vaultWith(res, base, settings, search)
}

func plugins(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources) (pluginInstall, settingsWiring, searchWiring, vaultWiring) {
	return assemble(c, exec, prefs, endpoints, res, nil)
}

func openIndexedPlugins(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, db *Index) (pluginInstall, settingsWiring, searchWiring, vaultWiring) {
	return assemble(c, exec, prefs, endpoints, res, newTextVersions(vaultBase(c, getRunningExecution(exec, c.SafeMode), getPluginEndpoints(endpoints), res, nil), db))
}

func setVaultSearch(v vaultWiring, search searchWiring) vaultWiring {
	v.CutNotes = cutNotes(search)
	v.SetNoteCut = func(scan *vault.Scan) { setNoteCut(search, scan) }
	return v
}

func openIndexedVault(search searchWiring, v vaultWiring, db *Index) (searchWiring, vaultWiring) {
	v.TextVersions = newTextVersions(v, db)
	search.TextReaders = getTextReaders(v)
	return search, setVaultSearch(v, search)
}
