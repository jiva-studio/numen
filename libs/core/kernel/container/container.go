// Package container is the composition root: the only place that selects which
// concrete adapter satisfies which port.
//
// Use cases above it work with the interfaces in core/port. To replace the index or
// the vault reader, only this package changes.
package container

import (
	"context"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
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
	ResolvedCompute settingsdomain.ResolvedCompute
	Hardware        settingsdomain.MachineSpecs
}

// Config is what the user may point somewhere else. Empty fields mean the platform's own locations. What the
// parts of one installation share and what the settings and the open vault give it is in Resources.
type Config struct {
	Paths

	PluginHost *PluginHost

	PluginCatalog catalog.Config

	KeyStore pluginport.KeyStore

	Credentials pluginport.CredentialsLocator

	TextVersions *TextVersions

	ShouldRebuildIndex bool

	SafeMode bool

	ErrorHandler window.ErrorHandler

	Now machineport.Clock

	Resources *Resources
}

// Resources is what every part a container assembles shares: the write lock of each vault, the
// folder watches, the access to the settings file and the state the settings give the installation. An
// application builds one with NewResources and names it in its Config. Chunks is the chunk size the embedding
// plugin reports, which the notes are cut at; TextReaders read the text of the files of a vault for an
// installation that has no text versions to read it from.
type Resources struct {
	Chunks      *note.ChunkTokens
	TextReaders textdomain.Readers

	machine         Machine
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
		Chunks:        &note.ChunkTokens{},
		vaultLocks:    vaultwriter.NewLocks(),
		watches:       vaultwatch.NewHub(),
		settingsFiles: settingsadapter.NewFiles(),
		settingEvents: settingevents.New(),
		pluginChanges: &settingsusecase.PluginChanges{},
		secretGate:    secret.NewGate(),
		programs:      &settingsadapter.ProgramList{},
	}
}

// NewConfig returns c with the resources of one container, and its plugin host in safe mode or out of it as
// SafeMode says. A Config built without NewConfig and without Resources opens nothing. The plugin host of a
// Config that names none is one that runs nothing.
func NewConfig(c Config) Config {
	if c.Resources == nil {
		c.Resources = NewResources()
	}
	c.PluginHost = c.host()
	c.PluginHost.Runtime.SetSafeMode(c.SafeMode)
	return c
}

// GetMachine is what the hardware allows the installation to use, as the settings last resolved it.
func (c Config) GetMachine() Machine { return c.Resources.machine }

func (c Config) host() *PluginHost {
	if c.PluginHost == nil {
		return newAbsentHost(plugindomain.NotInstalled, nil)
	}
	return c.PluginHost
}

func (c Config) runningHost() *PluginHost {
	if c.SafeMode {
		return newAbsentHost(plugindomain.Disabled, errSafeMode)
	}
	return c.host()
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
	return clock.System{}.Wait
}

func settingsOf(c Config) settingsWiring {
	res := c.Resources
	return settingsWiring{
		settingsPlaces: settingsPlaces{
			SettingsPath:    c.SettingsPath,
			ThemesPath:      c.ThemesPath,
			RegistryPath:    c.RegistryPath,
			ResolvedCompute: res.machine.ResolvedCompute,
		},
		settingsStores: settingsStores{
			Watches:         res.watches,
			SettingsFiles:   res.settingsFiles,
			SettingEvents:   res.settingEvents,
			Programs:        res.programs,
			SecretGate:      res.secretGate,
			PluginChanges:   res.pluginChanges,
			SubscribedHosts: &res.subscribedHosts,
		},
		PluginHost:   c.host(),
		ErrorHandler: func(err error) { handleError(c, err) },
	}
}

func searchOf(c Config) searchWiring {
	v := vaultBase(c)
	return searchWiring{
		ServiceDir:         c.ServiceDir,
		ShouldRebuildIndex: c.ShouldRebuildIndex,
		Chunks:             c.Resources.Chunks,
		ResolvedCompute:    c.Resources.machine.ResolvedCompute,
		PluginHost:         c.host(),
		RunningHost:        c.runningHost(),
		Clock:              c.Clock(),
		VaultReaders:       vaultReaders(v),
		TextReaders:        getTextReaders(v),
		ErrorHandler:       func(err error) { handleError(c, err) },
	}
}

func vaultBase(c Config) vaultWiring {
	res := c.Resources
	return vaultWiring{
		IndexPath:    c.IndexPath,
		RegistryPath: c.RegistryPath,
		ServiceDir:   c.ServiceDir,
		Clock:        c.Clock(),
		PluginHost:   c.host(),
		RunningHost:  c.runningHost(),
		TextVersions: c.TextVersions,
		TextReaders:  res.TextReaders,
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
	v.SetNoteCut = func(scan *vault.Scan) { setNoteCut(s, scan) }
	return v
}

func plugins(c Config) pluginsWiring {
	return pluginsWiring{
		pluginInstall: pluginInstall{
			PluginCachePath: c.PluginCachePath,
			ServiceDir:      c.ServiceDir,
			PluginHost:      c.host(),
			RunningHost:     c.runningHost(),
			PluginCatalog:   c.PluginCatalog,
			KeyStore:        c.KeyStore,
			CredentialsPath: c.getCredentialsPath(),
		},
		settings: settingsOf(c),
		vault:    vaultOf(c),
	}
}

func (c Config) agentPrograms(provider agentport.AgentProvider, target settingsadapter.Agent) []agentdomain.Program {
	return getAgentPrograms(plugins(c), provider, target)
}

func (c Config) getCredentialsPath() func(ctx context.Context) string {
	if c.Credentials == nil {
		return nil
	}
	return c.Credentials.GetCredentialsPath
}
