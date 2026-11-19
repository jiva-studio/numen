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

func vaultWith(res *Resources, base vaultWiring, settings settingsWiring, chunks noteChunkPolicy) vaultWiring {
	base.SyncSetting = syncSetting(settings)
	base.CutNotes = cutNotes(chunks)
	base.SetNoteCut = func(scan *vault.Scan) { setNoteCut(chunks, scan) }
	record := openRecordLog(base.IndexPath, getLogEntries(res), settings.SettingsFiles.GetLogLevel, base.Clock)
	base.RecordLog = recordLogFunc(record, res.ErrorHandler)
	return base
}

func assemble(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, texts *TextVersions) pluginsWiring {
	// texts are the text versions the open made over its index, or nil where it makes none.
	exec = getPluginExecution(exec)
	running := getRunningExecution(exec, c.SafeMode)
	endpoints = getPluginEndpoints(endpoints)
	settings := settingsOf(c, exec, prefs, res)
	base := vaultBase(c, running, endpoints, res, texts)
	vaultFiles, textReaders := vaultReaders(base), getTextReaders(base)
	chunks := noteChunkPolicy{Chunks: res.Chunks, Profiles: exec.Runtime.PluginResolver}
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
		search:   textSearch{VaultReaders: vaultFiles, TextReaders: textReaders, Workers: c.machine.ResolvedCompute.Workers},
		indexing: vaultIndexPlan{
			ChunkPolicy:        chunks,
			ServiceDir:         c.ServiceDir,
			ShouldRebuildIndex: c.ShouldRebuildIndex,
			VaultReaders:       vaultFiles,
			TextReaders:        textReaders,
			ErrorHandler:       getErrorHandler(c.IndexPath, res),
		},
		source: sourceBinding{
			Activator: running.Runtime.Activator,
			Plugins: sourcePlugins{
				PluginResolver: running.Runtime.PluginResolver,
				ConfigWatcher:  running.Runtime.ConfigWatcher,
			},
			Descriptions: endpoints.SourceDescriptions,
			Health:       exec.Health,
			Chunks:       res.Chunks,
		},
		vault: vaultWith(res, base, settings, chunks),
	}
}

func plugins(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources) pluginsWiring {
	return assemble(c, exec, prefs, endpoints, res, nil)
}

func openIndexedPlugins(c Config, exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, db *Index) pluginsWiring {
	return assemble(c, exec, prefs, endpoints, res, newTextVersions(vaultBase(c, getRunningExecution(exec, c.SafeMode), getPluginEndpoints(endpoints), res, nil), db))
}
