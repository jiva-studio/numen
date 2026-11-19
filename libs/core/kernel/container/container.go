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
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
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

func (c Config) searchWith(exec, running *PluginExecution, endpoints *PluginEndpoints, res *Resources, readers vaultport.VaultReaders, texts textdomain.Readers) searchWiring {
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
		VaultReaders:       readers,
		TextReaders:        texts,
		ErrorHandler:       getErrorHandler(c.IndexPath, res),
	}
}

func assemble(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, texts *TextVersions) pluginsWiring {
	// texts are the text versions the open made over its index, or nil where it makes none.
	exec = getPluginExecution(exec)
	running := getRunningExecution(exec, c.SafeMode)
	endpoints = getPluginEndpoints(endpoints)
	settings := settingsOf(c, exec, prefs, res)
	access := openVaultAccess(c.Paths, GetClock(res), res.vaultLocks, res.watches, texts)
	search := c.searchWith(exec, running, endpoints, res, access.Readers, getTextReaders(res.TextReaders, texts))
	access.SyncSetting = syncSetting(settings)
	access.CutNotes = cutNotes(search)
	access.SetNoteCut = func(scan *vault.Scan) { setNoteCut(search, scan) }
	record := openRecordLog(c.IndexPath, getLogEntries(res), settings.SettingsFiles.GetLogLevel, access.Clock)
	access.RecordLog = recordLogFunc(record, res.ErrorHandler)
	return pluginsWiring{
		pluginInstall: pluginInstall{
			PluginCachePath: c.PluginCachePath,
			ServiceDir:      c.ServiceDir,
			Execution:       exec,
			ActiveExecution: running,
			Endpoints:       endpoints,
			PluginCatalog:   res.PluginCatalog,
			KeyStore:        res.KeyStore,
		},
		settings: settings,
		search:   search,
		vault:    access,
	}
}

func plugins(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources) pluginsWiring {
	return assemble(c, exec, prefs, endpoints, res, nil)
}

func openIndexedPlugins(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, db *Index) pluginsWiring {
	running := getRunningExecution(exec, c.SafeMode)
	texts := newTextVersions(vaultconfig.Options{ServiceDir: c.ServiceDir}, running.Runtime.TextPlugins, GetClock(res), db)
	return assemble(c, exec, prefs, endpoints, res, texts)
}
