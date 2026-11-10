// Package container is the composition root: the only place that selects which
// concrete adapter satisfies which port.
//
// Use cases above it work with the interfaces in core/port. To replace the index or
// the vault reader, only this package changes.
package container

import (
	"context"
	"time"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
)

// Config is what the user may point somewhere else. Empty fields mean the
// platform's own locations.
type Config struct {
	IndexPath       string
	RegistryPath    string
	ServiceDir      string
	SettingsPath    string
	ThemesPath      string
	PluginCachePath string

	InterfaceScale, TextScale float64

	ResolvedCompute hardware.ResolvedCompute

	Hardware hardware.Specs

	Chunks *note.ChunkTokens

	PluginHost *PluginHost

	PluginCatalog catalog.Config

	PluginDirs []string

	KeyStore pluginport.KeyStore

	TextVersions *textversion.Versions

	TextReaders textdomain.Readers

	ShouldRebuildIndex bool

	SafeMode bool

	ErrorHandler window.ErrorHandler

	Now machineport.Clock

	Resources *Resources

	agentProvider agentport.AgentProvider
	programs      *settingsadapter.ProgramList
}

// Resources is what every part a container assembles shares: the write lock of each vault, the
// folder watches and the access to the settings file. An application builds one with
// NewResources and names it in its Config.
type Resources struct {
	vaultLocks      *vaultwriter.Locks
	watches         *vaultwatch.Hub
	settingsFiles   *settingsadapter.Files
	settingEvents   *settingevents.Events
	secretGate      *secret.Gate
	pluginChanges   *settingsusecase.PluginChanges
	programs        *settingsadapter.ProgramList
	subscribedHosts settingsusecase.HostSubscriptions
}

// NewResources returns the resources of one container.
func NewResources() *Resources {
	return &Resources{
		vaultLocks:    vaultwriter.NewLocks(),
		watches:       vaultwatch.NewHub(),
		settingsFiles: settingsadapter.NewFiles(),
		settingEvents: settingevents.New(),
		pluginChanges: &settingsusecase.PluginChanges{},
		secretGate:    secret.NewGate(),
		programs:      &settingsadapter.ProgramList{},
	}
}

// NewConfig returns c with the resources of one container. A Config built without NewConfig and
// without Resources opens writers, watches and settings that refuse to work.
func NewConfig(c Config) Config {
	if c.Resources == nil {
		c.Resources = NewResources()
	}
	if c.Chunks == nil {
		c.Chunks = &note.ChunkTokens{}
	}
	return c
}

func getResources(c Config) *Resources {
	if c.Resources != nil {
		return c.Resources
	}
	return &Resources{vaultLocks: nil, watches: nil, settingsFiles: nil, settingEvents: nil, secretGate: nil, pluginChanges: &settingsusecase.PluginChanges{}}
}

func getPrograms(c Config) *settingsadapter.ProgramList {
	if c.programs != nil {
		return c.programs
	}
	return getResources(c).programs
}

// Clock is the clock every scenario assembled here is handed.
func (c Config) Clock() machineport.Clock {
	if c.Now != nil {
		return c.Now
	}
	return clock.System{}
}

func handleError(c Config, err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
}

// GetWait is the wait every scenario assembled here is handed, bound to this
// machine's timers.
func (c Config) GetWait() machine.Wait {
	return func(ctx context.Context, d time.Duration) error {
		if d <= 0 {
			return ctx.Err()
		}
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
}

func settingsOf(c Config) settingsWiring {
	res := getResources(c)
	return settingsWiring{
		SettingsPath:     c.SettingsPath,
		ThemesPath:       c.ThemesPath,
		RegistryPath:     c.RegistryPath,
		InterfaceScale:   c.InterfaceScale,
		TextScale:        c.TextScale,
		Hardware:         c.Hardware,
		ResolvedCompute:  c.ResolvedCompute,
		PluginHost:       c.PluginHost,
		KeyStore:         c.KeyStore,
		Watches:          res.watches,
		SettingsFiles:    res.settingsFiles,
		SettingEvents:    res.settingEvents,
		Programs:         getPrograms(c),
		SecretGate:       res.secretGate,
		PluginChanges:    res.pluginChanges,
		SubscribedHosts:  &res.subscribedHosts,
		ErrorHandler:     func(err error) { handleError(c, err) },
		AgentProvider:    func() agentport.AgentProvider { return pluginsAgentProvider(plugins(c)) },
		GetAgentPrograms: func(p agentport.AgentProvider, t settingsadapter.Agent) []agentdomain.Program { return getAgentPrograms(plugins(c), p, t) },
		SetSecretSource:  func() { setSecretSource(plugins(c)) },
		SetCompleter:     func() { setCompleter(plugins(c)) },
	}
}

func searchOf(c Config) searchWiring {
	v := vaultBase(c)
	plugins := plugins(c)
	return searchWiring{
		ServiceDir:         c.ServiceDir,
		ShouldRebuildIndex: c.ShouldRebuildIndex,
		Chunks:             c.Chunks,
		ResolvedCompute:    c.ResolvedCompute,
		PluginHost:         c.PluginHost,
		Clock:              c.Clock(),
		VaultReaders:       vaultReaders(v),
		TextReaders:        getTextReaders(v),
		ErrorHandler:       func(err error) { handleError(c, err) },
		ResolvePlugin: func(capability string) (*PluginHost, error) {
			return pluginsResolvePlugin(plugins, capability)
		},
	}
}

func vaultBase(c Config) vaultWiring {
	res := getResources(c)
	return vaultWiring{
		IndexPath:    c.IndexPath,
		RegistryPath: c.RegistryPath,
		ServiceDir:   c.ServiceDir,
		SafeMode:     c.SafeMode,
		Clock:        c.Clock(),
		PluginHost:   c.PluginHost,
		TextVersions: c.TextVersions,
		TextReaders:  c.TextReaders,
		VaultLocks:   res.vaultLocks,
		Watches:      res.watches,
		ErrorHandler: func(err error) { handleError(c, err) },
	}
}

func vaultOf(c Config) vaultWiring {
	v := vaultBase(c)
	s := searchOf(c)
	v.SyncSetting = syncSetting(settingsOf(c))
	v.CutNotes = cutNotes(s)
	v.SetNoteCut = func(scan *vault.Scan, db *vault.Index) { setNoteCut(s, scan, db) }
	return v
}

func plugins(c Config) pluginsWiring {
	res := getResources(c)
	programs := getPrograms(c)
	v := vaultBase(c)
	s := settingsWiring{
		SettingsPath:  c.SettingsPath,
		ThemesPath:    c.ThemesPath,
		RegistryPath:  c.RegistryPath,
		PluginHost:    c.PluginHost,
		Watches:       res.watches,
		SettingsFiles: res.settingsFiles,
		SettingEvents: res.settingEvents,
		Programs:      programs,
		ErrorHandler:  func(err error) { handleError(c, err) },
	}
	return pluginsWiring{
		PluginCachePath:       c.PluginCachePath,
		ServiceDir:            c.ServiceDir,
		SafeMode:              c.SafeMode,
		PluginHost:            c.PluginHost,
		PluginCatalog:         c.PluginCatalog,
		PluginDirs:            c.PluginDirs,
		KeyStore:              c.KeyStore,
		SettingsFiles:         res.settingsFiles,
		SecretGate:            res.secretGate,
		PluginChanges:         res.pluginChanges,
		TextVersions:          c.TextVersions,
		ListAppSettings:       func() []plugindomain.Setting { return listAppSettings(s) },
		CredentialsPath:       func() string { return getCredentialsPath(plugins(c)) },
		Programs:              programs,
		VaultReaders:          vaultReaders(v),
		NoteWriters:           noteWriters(v),
		VaultLocks:            writeLocks(v),
		VaultWatcher:          vaultWatcher(v),
		ErrorHandler:          func(err error) { handleError(c, err) },
		SettingsFile:          func() (string, error) { return settingsSettingsFile(s) },
		ReadCheckedSettings:   func() (settingsadapter.File, error) { return readCheckedSettings(s) },
		GetSettingsOrDefaults: func() settingsadapter.File { return settingsGetSettingsOrDefaults(s) },
		RefreshSettingEvents:  func() { refreshSettingEvents(s) },
		MakeRefresh: func(db *vault.Index, notes noteport.NoteRepository) vault.Refresh {
			return vaultMakeRefresh(vaultOf(c), db, notes)
		},
		BuildNotes: func(db *vault.Index, index edit.IndexUpdater) note.Scenarios { return Notes(c, db, index) },
		Moves:      func(ctx context.Context) (<-chan struct{}, error) { return vaultMoves(v, ctx) },
	}
}
