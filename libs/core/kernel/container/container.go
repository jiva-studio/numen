// Package container is the composition root: the only place that selects which
// concrete adapter satisfies which port.
//
// Use cases above it work with the interfaces in core/port. To replace the index or
// the vault reader, only this package changes.
package container

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/vaultabsent"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"

	windowlogs "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/logs"

	machinedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/machine"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"

	keyfileusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/keyfile"
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
	KeyFile       keyfileusecase.File
	Stores        identity.Stores
	ErrorHandler  window.ErrorHandler
	Now           identity.Clock
	Chunks        *note.ChunkTokens
	TextReaders   textport.Readers

	vault         vaultShared
	settingFiles  *settingsadapter.Files
	settingEvents *settingevents.Events
	subscriptions settingsusecase.Subscriptions
	logEntries    *logsusecase.Entries
	progress      *identity.Progress
	absentFiles   vaultport.AbsentFiles
}

// vaultShared holds what every vault of one installation shares: the write locks and the watches.
type vaultShared struct {
	locks   *vaultwriter.Locks
	watches *vaultwatch.Hub
}

func NewResources() *Resources {
	return &Resources{
		Chunks:        &note.ChunkTokens{},
		vault:         vaultShared{locks: vaultwriter.NewLocks(), watches: vaultwatch.NewHub()},
		settingFiles:  settingsadapter.NewFiles(),
		settingEvents: settingevents.New(),
		subscriptions: settingsusecase.Subscriptions{Changes: settingsusecase.PluginChanges{}, Hosts: settingsusecase.HostSubscriptions{}},
		logEntries:    logsusecase.NewEntries(),
		progress:      identity.NewProgress(),
	}
}

func getCreateGate(res *Resources) identity.CreateGate {
	if res.progress == nil {
		return nil
	}
	return res.progress.GetCreateGate()
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
func GetClock(res *Resources) identity.Clock {
	if res.Now != nil {
		return res.Now
	}
	return clock.System{}
}

func getErrorHandler(indexPath string, res *Resources) window.ErrorHandler {
	return windowlogs.ReportError(openOptionalRecordLog(indexPath, res), res.ErrorHandler, "Background operation failed")
}

func createSettingsWiring(c Config, exec *PluginExecution, prefs *PluginPreferences, res *Resources) settingsParts {
	runtime := getPluginExecution(exec).Runtime
	prefs = getPluginPreferences(prefs)
	return settingsParts{
		SettingsPath:    c.SettingsPath,
		ThemesPath:      c.ThemesPath,
		RegistryPath:    c.RegistryPath,
		Watches:         res.vault.watches,
		SettingsFiles:   res.settingFiles,
		SettingEvents:   res.settingEvents,
		PluginChanges:   &res.subscriptions.Changes,
		Infos:           runtime.Listing,
		Declarations:    prefs.Configuration.pluginSettingDeclarer,
		ConnectionTypes: prefs.Views.Types,
		PluginInfos:     prefs.Views.Infos,
		Checks:          prefs.Views.Checks,
		Detection:       prefs.Detection,
		HandleError:     getErrorHandler(c.IndexPath, res),
	}
}

func buildVaultSearch(c Config, exec *PluginExecution, prefs *PluginPreferences, res *Resources, texts *textVersions) (settingsParts, vaultIndexPlan, vaultParts) {
	// texts are the text versions the open made over its index, or nil where it makes none.
	exec = getPluginExecution(exec)
	settings := createSettingsWiring(c, exec, prefs, res)
	base := openVaultAccess(c.Paths, GetClock(res), res.vault.locks, res.vault.watches, texts)
	base.Readers, base.AbsentFiles = vaultabsent.Wrap(base.Readers, res.absentFiles), res.absentFiles
	plan := vaultIndexPlan{
		ChunkPolicy: noteChunkPolicy{Chunks: res.Chunks, Profiles: exec.Runtime.Resolver},
		ServiceDir:  c.ServiceDir, ShouldRebuildIndex: c.ShouldRebuildIndex,
		VaultReaders: base.Readers, TextReaders: getTextReaders(res.TextReaders, texts), HandleError: settings.HandleError,
	}
	base.SyncSetting = syncSetting(settings)
	base = setVaultSearch(base, plan.ChunkPolicy)
	base.RecordLog = openOptionalRecordLog(base.IndexPath, res)
	return settings, plan, base
}

func setVaultSearch(v vaultParts, chunks noteChunkPolicy) vaultParts {
	v.CutNotes = cutNotes(chunks)
	v.SetNoteCut = func(scan *indexing.Scan) { setNoteCut(chunks, scan) }
	return v
}

func openIndexedVault(plan vaultIndexPlan, v vaultParts, db *vaultIndex, plugins textversion.TextPlugins) (vaultIndexPlan, vaultParts) {
	v = openVaultTextVersions(v, db, plugins, plan.ChunkPolicy)
	plan.TextReaders = getTextReaders(plan.TextReaders, v.TextVersions)
	return plan, v
}

func openVaultTextVersions(v vaultParts, db *vaultIndex, plugins textversion.TextPlugins, chunks noteChunkPolicy) vaultParts {
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
		Keys:            identity.SharedKeys{Store: res.Stores.Keys, CreateLock: getCreateGate(res)},
	}
}

type timeSource interface {
	Now() time.Time
	StartTimer(d time.Duration) <-chan time.Time
}
