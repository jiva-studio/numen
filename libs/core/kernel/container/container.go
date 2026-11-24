// Package container is the composition root: the only place that selects which
// concrete adapter satisfies which port.
//
// Use cases above it work with the interfaces in core/port. To replace the index or
// the vault reader, only this package changes.
package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/vaultabsent"

	windowlogs "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/logs"

	machinedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/machine"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"

	logsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"

	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
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
	Absent             bool
	Scales             Scales
	machine            Machine
}

// Resources holds the services shared by the parts of one installation.
type Resources struct {
	PluginCatalog catalog.Config
	KeyStore      pluginport.KeyStore
	ChainStore    identity.ChainStore
	ErrorHandler  window.ErrorHandler
	Now           installationClock
	Chunks        *note.ChunkTokens
	TextReaders   textport.Readers

	vaultLocks   *vaultwriter.Locks
	watches      *vaultwatch.Hub
	settings     settingsResources
	logEntries   *logsusecase.Entries
	progress     *identity.Progress
	identityLock identity.CreateGate
	absentFiles  vaultport.AbsentFiles
}

type settingsResources struct {
	files           *settingsadapter.Files
	events          *settingevents.Events
	pluginChanges   *settingsusecase.PluginChanges
	subscribedHosts settingsusecase.HostSubscriptions
}

func NewResources() *Resources {
	progress := identity.NewProgress()
	return &Resources{
		Chunks:     &note.ChunkTokens{},
		vaultLocks: vaultwriter.NewLocks(),
		watches:    vaultwatch.NewHub(),
		settings: settingsResources{
			files:           settingsadapter.NewFiles(),
			events:          settingevents.New(),
			pluginChanges:   &settingsusecase.PluginChanges{},
			subscribedHosts: settingsusecase.HostSubscriptions{},
		},
		logEntries:   logsusecase.NewEntries(),
		progress:     progress,
		identityLock: progress.GetCreateGate(),
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
func GetClock(res *Resources) installationClock {
	if res.Now != nil {
		return res.Now
	}
	return clock.System{}
}

func getErrorHandler(indexPath string, res *Resources) window.ErrorHandler {
	return windowlogs.ReportError(openOptionalRecordLog(indexPath, res), res.ErrorHandler, "Background operation failed")
}

func createSettingsWiring(c Config, exec *PluginExecution, prefs *PluginPreferences, res *Resources) settingsWiring {
	runtime := getPluginExecution(exec).Runtime
	prefs = getPluginPreferences(prefs)
	return settingsWiring{
		SettingsPath:    c.SettingsPath,
		ThemesPath:      c.ThemesPath,
		RegistryPath:    c.RegistryPath,
		Watches:         res.watches,
		SettingsFiles:   res.settings.files,
		SettingEvents:   res.settings.events,
		PluginChanges:   res.settings.pluginChanges,
		Infos:           runtime.Listing,
		Declarations:    prefs.Configuration.pluginSettingDeclarer,
		ConnectionTypes: prefs.Views.Types,
		PluginInfos:     prefs.Views.Infos,
		Checks:          prefs.Views.Checks,
		Detection:       prefs.Detection,
		HandleError:     getErrorHandler(c.IndexPath, res),
	}
}

func createVaultWiring(res *Resources, base vaultWiring, settings settingsWiring, chunks noteChunkPolicy) vaultWiring {
	base.SyncSetting = syncSetting(settings)
	base = setVaultSearch(base, chunks)
	base.RecordLog = openOptionalRecordLog(base.IndexPath, res)
	return base
}

func buildVaultSearch(c Config, exec *PluginExecution, prefs *PluginPreferences, res *Resources, texts *textVersions) (settingsWiring, vaultIndexPlan, vaultWiring) {
	// texts are the text versions the open made over its index, or nil where it makes none.
	exec = getPluginExecution(exec)
	settings := createSettingsWiring(c, exec, prefs, res)
	base := openVaultAccess(c.Paths, GetClock(res), res.vaultLocks, res.watches, texts)
	base.Readers, base.AbsentFiles = vaultabsent.Wrap(base.Readers, res.absentFiles), res.absentFiles
	plan := vaultIndexPlan{
		ChunkPolicy: noteChunkPolicy{Chunks: res.Chunks, Profiles: exec.Runtime.Resolver},
		ServiceDir:  c.ServiceDir, ShouldRebuildIndex: c.ShouldRebuildIndex,
		VaultReaders: base.Readers, TextReaders: getTextReaders(res.TextReaders, texts), HandleError: settings.HandleError,
	}
	return settings, plan, createVaultWiring(res, base, settings, plan.ChunkPolicy)
}

func setVaultSearch(v vaultWiring, chunks noteChunkPolicy) vaultWiring {
	v.CutNotes = cutNotes(chunks)
	v.SetNoteCut = func(scan *indexing.Scan) { setNoteCut(chunks, scan) }
	return v
}

func openIndexedVault(plan vaultIndexPlan, v vaultWiring, db *vaultIndex, plugins textversion.TextPlugins) (vaultIndexPlan, vaultWiring) {
	v = openVaultTextVersions(v, db, plugins, plan.ChunkPolicy)
	plan.TextReaders = getTextReaders(plan.TextReaders, v.TextVersions)
	return plan, v
}

func openVaultTextVersions(v vaultWiring, db *vaultIndex, plugins textversion.TextPlugins, chunks noteChunkPolicy) vaultWiring {
	v.TextVersions = newTextVersions(v.Options, plugins, v.Clock, db)
	return setVaultSearch(v, chunks)
}

func newSourceBinding(exec *PluginExecution, safeMode bool, endpoints *PluginEndpoints, res *Resources) sourcebridge.Binding {
	runtime := getRunningExecution(exec, safeMode).Runtime
	return sourcebridge.Binding{
		Activator:    runtime.Activator,
		Plugins:      sourcebridge.PluginAccess{CapabilityResolver: runtime.Resolver, ConfigWatcher: runtime.ConfigWatcher},
		Descriptions: getPluginEndpoints(endpoints).SourceDescriptions,
		ReportRead:   pluginhealth.NewReportSearchSource(exec.Health).Execute,
		Follow:       res.Chunks.Follow,
	}
}

func newPluginInstall(c Config, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources) pluginInstall {
	prefs = getPluginPreferences(prefs)
	return pluginInstall{
		Licenses: prefs.Views.Declarations, Configuration: prefs.Configuration, OperationTitles: prefs.Views.Operations,
		PluginCachePath: c.PluginCachePath,
		ServiceDir:      c.ServiceDir,
		SafeMode:        c.SafeMode,
		Endpoints:       getPluginEndpoints(endpoints),
		PluginCatalog:   res.PluginCatalog,
		Keys:            identity.SharedKeys{Store: res.KeyStore, CreateLock: res.identityLock},
	}
}
