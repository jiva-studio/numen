// Package container is the composition root: the only place that selects which
// concrete adapter satisfies which port.
//
// Use cases above it work with the interfaces in core/port. To replace the index or
// the vault reader, only this package changes.
package container

import (
	"context"
	"sync"
	"time"

	settingsadapter "github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"
	connectionswiring "github.com/jiva-studio/numen/libs/core/container/connections"
	pluginswiring "github.com/jiva-studio/numen/libs/core/container/plugins"
	searchwiring "github.com/jiva-studio/numen/libs/core/container/search"
	settingswiring "github.com/jiva-studio/numen/libs/core/container/settings"
	vaultwiring "github.com/jiva-studio/numen/libs/core/container/vault"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/pluginchanges"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
	"github.com/jiva-studio/numen/libs/core/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
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

	Chunks *ChunkTokens

	PluginHost pluginport.PluginManager

	PluginCatalog catalog.Config

	PluginDirs []string

	KeyStore pluginport.KeyStore

	TextVersions *textversion.Versions

	TextReaders textport.TextReaders

	ShouldRebuildIndex bool

	SafeMode bool

	ErrorHandler windowport.ErrorHandler

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
	pluginChanges   *pluginchanges.Signals
	subscribedHosts sync.Map
}

// NewResources returns the resources of one container.
func NewResources() *Resources {
	return &Resources{
		vaultLocks:    vaultwriter.NewLocks(),
		watches:       vaultwatch.NewHub(),
		settingsFiles: settingsadapter.NewFiles(),
		settingEvents: settingevents.New(),
		pluginChanges: &pluginchanges.Signals{},
		secretGate:    secret.NewGate(),
	}
}

// NewConfig returns c with the resources of one container. A Config built without NewConfig and
// without Resources opens writers, watches and settings that refuse to work.
func NewConfig(c Config) Config {
	if c.Resources == nil {
		c.Resources = NewResources()
	}
	if c.Chunks == nil {
		c.Chunks = &ChunkTokens{}
	}
	return c
}

func (c Config) getResources() *Resources {
	if c.Resources != nil {
		return c.Resources
	}
	return &Resources{vaultLocks: nil, watches: nil, settingsFiles: nil, settingEvents: nil, secretGate: nil, pluginChanges: &pluginchanges.Signals{}}
}

// Clock is the clock every scenario assembled here is handed.
func (c Config) Clock() machineport.Clock {
	if c.Now != nil {
		return c.Now
	}
	return clock.System{}
}

func (c Config) handleError(err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
}

// GetWait is the wait every scenario assembled here is handed, bound to this
// machine's timers.
func (c Config) GetWait() machineport.Wait {
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

// GetDelay is the delayed call every scenario assembled here is handed, bound
// to this machine's timers.
func (c Config) GetDelay() machineport.Delay {
	return func(d time.Duration, f func()) func() bool {
		return time.AfterFunc(d, f).Stop
	}
}

func (c Config) settings() settingswiring.Config {
	res := c.getResources()
	return settingswiring.Config{
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
		SecretGate:       res.secretGate,
		PluginChanges:    res.pluginChanges,
		SubscribedHosts:  &res.subscribedHosts,
		ErrorHandler:     c.handleError,
		AgentProvider:    c.AgentProvider,
		GetAgentPrograms: c.plugins().GetAgentPrograms,
		SetSecretSource:  c.plugins().SetSecretSource,
		SetCompleter:     c.plugins().SetCompleter,
	}
}

func (c Config) connections() connectionswiring.Config {
	res := c.getResources()
	s := settingswiring.Config{
		SettingsPath:  c.SettingsPath,
		ThemesPath:    c.ThemesPath,
		RegistryPath:  c.RegistryPath,
		PluginHost:    c.PluginHost,
		Watches:       res.watches,
		SettingsFiles: res.settingsFiles,
		SettingEvents: res.settingEvents,
		ErrorHandler:  c.handleError,
	}
	return connectionswiring.Config{
		PluginHost:           c.PluginHost,
		KeyStore:             c.KeyStore,
		SettingsFiles:        res.settingsFiles,
		SettingsFile:         s.SettingsFile,
		ReadCheckedSettings:  s.ReadCheckedSettings,
		RefreshSettingEvents: s.RefreshSettingEvents,
	}
}

// OpenConnections returns the use cases on the connections to services and programs, or nil where
// the installation has no plugin host.
func (c Config) OpenConnections() *connection.Connections {
	return c.connections().OpenConnections()
}

func (c Config) search() searchwiring.Config {
	v := c.vaultBase()
	return searchwiring.Config{
		ServiceDir:         c.ServiceDir,
		ShouldRebuildIndex: c.ShouldRebuildIndex,
		Chunks:             c.Chunks,
		ResolvedCompute:    c.ResolvedCompute,
		PluginHost:         c.PluginHost,
		Clock:              c.Clock(),
		VaultReaders:       v.VaultReaders(),
		TextReaders:        v.GetTextReaders(),
		ErrorHandler:       c.handleError,
		ResolvePlugin:      c.plugins().ResolvePlugin,
	}
}

func (c Config) vaultBase() vaultwiring.Config {
	res := c.getResources()
	return vaultwiring.Config{
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
		ErrorHandler: c.handleError,
	}
}

func (c Config) vault() vaultwiring.Config {
	v := c.vaultBase()
	s := c.search()
	v.SyncSetting = c.settings().SyncSetting()
	v.CutNotes = s.CutNotes()
	v.SetNoteCut = s.SetNoteCut
	return v
}

func (c Config) plugins() pluginswiring.Config {
	res := c.getResources()
	v := c.vaultBase()
	s := settingswiring.Config{
		SettingsPath:  c.SettingsPath,
		ThemesPath:    c.ThemesPath,
		RegistryPath:  c.RegistryPath,
		PluginHost:    c.PluginHost,
		Watches:       res.watches,
		SettingsFiles: res.settingsFiles,
		SettingEvents: res.settingEvents,
		ErrorHandler:  c.handleError,
	}
	return pluginswiring.Config{
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
		ListAppSettings:       s.ListAppSettings,
		Programs:              c.programs,
		VaultReaders:          v.VaultReaders(),
		VaultWriters:          v.VaultWriters(),
		VaultWatcher:          v.VaultWatcher(),
		ErrorHandler:          c.handleError,
		SettingsFile:          s.SettingsFile,
		ReadCheckedSettings:   s.ReadCheckedSettings,
		GetSettingsOrDefaults: s.GetSettingsOrDefaults,
		RefreshSettingEvents:  s.RefreshSettingEvents,
		MakeRefresh:           func(db *Index, notes noteRepository) vaultRefresh { return c.vault().MakeRefresh(db, notes) },
		BuildNotes:            c.Notes,
		Moves:                 v.Moves,
	}
}
