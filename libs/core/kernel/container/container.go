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

// PreparePluginHost selects the installation's host and applies its launch mode.
func PreparePluginHost(host *PluginHost, safeMode bool) *PluginHost {
	host = getPluginHost(host)
	host.Runtime.SetSafeMode(safeMode)
	return host
}

// GetMachine is what the hardware allows the installation to use, as the settings last resolved it.
func (c Config) GetMachine() Machine { return c.machine }

func getPluginHost(host *PluginHost) *PluginHost {
	if host == nil {
		return newAbsentHost(plugindomain.NotInstalled, nil)
	}
	return host
}

func getRunningHost(host *PluginHost, safeMode bool) *PluginHost {
	if safeMode {
		return newAbsentHost(plugindomain.Disabled, errSafeMode)
	}
	return getPluginHost(host)
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

func settingsOf(c Config, host *PluginHost, res *Resources) settingsWiring {
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
		PluginHost:   getPluginHost(host),
		ErrorHandler: getErrorHandler(c.IndexPath, res),
	}
}

func searchWith(c Config, res *Resources, v vaultWiring) searchWiring {
	return searchWiring{
		ServiceDir:         c.ServiceDir,
		ShouldRebuildIndex: c.ShouldRebuildIndex,
		Chunks:             res.Chunks,
		ResolvedCompute:    c.machine.ResolvedCompute,
		PluginHost:         v.PluginHost,
		RunningHost:        v.RunningHost,
		Clock:              GetClock(res),
		VaultReaders:       vaultReaders(v),
		TextReaders:        getTextReaders(v),
		ErrorHandler:       getErrorHandler(c.IndexPath, res),
	}
}

func vaultBase(c Config, host *PluginHost, res *Resources, texts *TextVersions) vaultWiring {
	return vaultWiring{
		IndexPath:    c.IndexPath,
		RegistryPath: c.RegistryPath,
		ServiceDir:   c.ServiceDir,
		Clock:        GetClock(res),
		PluginHost:   getPluginHost(host),
		RunningHost:  getRunningHost(host, c.SafeMode),
		TextVersions: texts,
		TextReaders:  res.TextReaders,
		VaultLocks:   res.vaultLocks,
		Watches:      res.watches,
		ErrorHandler: getErrorHandler(c.IndexPath, res),
	}
}

func vaultWith(res *Resources, base vaultWiring, settings settingsWiring, search searchWiring) vaultWiring {
	base.SyncSetting = syncSetting(settings)
	base.CutNotes = cutNotes(search)
	base.SetNoteCut = func(scan *vault.Scan) { setNoteCut(search, scan) }
	record := openRecordLog(base.IndexPath, getLogEntries(res), settings.SettingsFiles.GetLogLevel, base.Clock)
	base.RecordLog = recordLogFunc(record, res.ErrorHandler)
	return base
}

func assemble(c Config, host *PluginHost, res *Resources, texts *TextVersions) pluginsWiring {
	// texts are the text versions the open made over its index, or nil where it makes none.
	host = getPluginHost(host)
	settings := settingsOf(c, host, res)
	base := vaultBase(c, host, res, texts)
	search := searchWith(c, res, base)
	return pluginsWiring{
		pluginInstall: pluginInstall{
			PluginCachePath: c.PluginCachePath,
			ServiceDir:      c.ServiceDir,
			PluginHost:      host,
			RunningHost:     base.RunningHost,
			PluginCatalog:   res.PluginCatalog,
			KeyStore:        res.KeyStore,
		},
		settings: settings,
		search:   search,
		vault:    vaultWith(res, base, settings, search),
	}
}

func plugins(c Config, host *PluginHost, res *Resources) pluginsWiring {
	return assemble(c, host, res, nil)
}

func vaultOf(c Config, host *PluginHost, res *Resources) vaultWiring {
	return plugins(c, host, res).vault
}

func openIndexedPlugins(c Config, host *PluginHost, res *Resources, db *Index) pluginsWiring {
	return assemble(c, host, res, newTextVersions(vaultBase(c, host, res, nil), db))
}
