// Package settings wires the settings file, setting change events and themes.
package settings

import (
	"context"
	"fmt"
	"path/filepath"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings/hardware"
	pluginswiring "github.com/jiva-studio/numen/libs/core/kernel/container/plugins"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/pluginchanges"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// AgentAt is the key of the agent section in the settings file.
const AgentAt = "agent"

// Config wires the settings, setting change events and themes of an installation.
type Config struct {
	SettingsPath              string
	ThemesPath                string
	RegistryPath              string
	InterfaceScale, TextScale float64
	Hardware                  hardware.Specs
	ResolvedCompute           hardware.ResolvedCompute
	PluginHost                *pluginswiring.Host
	KeyStore                  pluginport.KeyStore
	Watches                   *vaultwatch.Hub
	SettingsFiles             *settingsadapter.Files
	SettingEvents             *settingevents.Events
	Programs                  *settingsadapter.ProgramList
	SecretGate                *secret.Gate
	SubscribedHosts           *settingsusecase.HostSubscriptions
	PluginChanges             *pluginchanges.Signals
	ErrorHandler              func(error)
	AgentProvider             func() agentport.AgentProvider
	GetAgentPrograms          func(agentport.AgentProvider, settingsadapter.Agent) []agent.Program
	SetSecretSource           func()
	SetCompleter              func()
}

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() settingsadapter.File {
	return settingsadapter.DefaultFile()
}

// Settings reads the settings file. An unconfigured installation returns the
// default settings.
func (c Config) Settings() (settingsadapter.File, error) {
	path, err := c.SettingsFile()
	if err != nil {
		return settingsadapter.File{}, err
	}
	return c.store().ReadCheckedAt(path)
}

// GetSettingsAt reads the settings file at path without filtering plugin declarations.
func (c Config) GetSettingsAt(path string) (settingsadapter.File, error) {
	return c.store().ReadUncheckedAt(path)
}

// GetSettingsOrDefaults returns the checked settings or DefaultSettings on failure.
func (c Config) GetSettingsOrDefaults() settingsadapter.File {
	if path, err := c.SettingsFile(); err == nil {
		if target, err := c.store().ReadCheckedAt(path); err == nil {
			return target
		}
	}
	return DefaultSettings()
}

// ReadCheckedSettings reads and validates the active settings file, or returns defaults if missing.
func (c Config) ReadCheckedSettings() (settingsadapter.File, error) {
	return c.store().ReadCheckedFile()
}

// ResolveCompute detects the machine hardware and resolves the compute plan for s.
func (c Config) ResolveCompute(s settingsadapter.File) (hardware.Specs, hardware.ResolvedCompute) {
	specs := hardware.Detect()
	return specs, hardware.ResolveCompute(s.Performance.Profile, specs)
}

// ConfigurePluginHost applies the settings s to the plugin host and subscribes it to changes.
func (c Config) ConfigurePluginHost(s settingsadapter.File) {
	if c.PluginHost == nil {
		return
	}
	hosts := c.SubscribedHosts
	if hosts == nil {
		hosts = &settingsusecase.HostSubscriptions{}
	}
	var subscribe settingsport.ChangeSubscriber
	if c.SettingEvents != nil {
		subscribe = c.feed().Subscribe
	}
	settingsusecase.NewApplyToPlugins(
		settingsadapter.JSONCodec{}, c.PluginHost, c.writeSettings(), c.notifyPlugins(), subscribe, hosts, c.setUpPluginHost,
	).Execute(s.Plugins, c.ResolvedCompute.Profile)
}

func (c Config) setUpPluginHost() {
	c.PluginHost.SetKeyStore(c.KeyStore)
	if c.SetSecretSource != nil {
		c.SetSecretSource()
	}
	if c.SetCompleter != nil {
		c.SetCompleter()
	}
}

// UnsubscribeHost ends the delivery of setting changes to the plugin host and returns after the
// batch being delivered has been handled.
func (c Config) UnsubscribeHost() {
	if c.SubscribedHosts == nil || c.PluginHost == nil {
		return
	}
	c.SubscribedHosts.Unsubscribe(c.PluginHost)
}

// NotifyHost forwards a batch of changed settings to the plugin host.
func (c Config) NotifyHost(batch []settingsdomain.Setting) {
	c.notifyPlugins().Execute(batch)
}

// Plugins returns a function that lists the discovered plugins and their
// settings schemas.
func (c Config) Plugins() func() []plugindomain.Info {
	return func() []plugindomain.Info {
		if c.PluginHost == nil {
			return nil
		}
		return c.PluginHost.GetPluginInfos()
	}
}

// Models returns a function that lists the models each model setting accepts,
// and the programs the agent setting accepts.
func (c Config) Models() func() []settingsdomain.Model {
	var provider agentport.AgentProvider
	if c.AgentProvider != nil {
		provider = c.AgentProvider()
	}
	return c.ModelsForProvider(provider)
}

// ModelsForProvider returns a function that lists the models for the given agent provider.
func (c Config) ModelsForProvider(provider agentport.AgentProvider) func() []settingsdomain.Model {
	return func() []settingsdomain.Model {
		target := c.GetSettingsOrDefaults()
		var programs []agent.Program
		if c.GetAgentPrograms != nil {
			programs = c.GetAgentPrograms(provider, target.Agent)
		}
		return PrefixModels(settingsadapter.OwnerApp, settingsadapter.AgentModels(AgentAt, target.Agent, programs))
	}
}

// PrefixModels prepends owner to the path of each model and its writes.
func PrefixModels(owner string, models []settingsdomain.Model) []settingsdomain.Model {
	for place := range models {
		models[place].Path = append([]string{owner}, models[place].Path...)
		for index := range models[place].Writes {
			models[place].Writes[index].Path = append([]string{owner}, models[place].Writes[index].Path...)
		}
	}
	return models
}

// GetDeclarations returns the setting declarations for the app and plugins.
func (c Config) GetDeclarations() settingsadapter.Declarations {
	return func(owner string) ([]manifest.Setting, bool) {
		if owner == settingsadapter.OwnerApp {
			return settingsadapter.GetAppDeclaration(), true
		}
		if c.PluginHost == nil {
			return nil, false
		}
		return c.PluginHost.GetPluginSettingDeclarations(owner)
	}
}

// GetRegistries returns the option values for a named setting registry.
func (c Config) GetRegistries() settingsadapter.Registries {
	return func(name string) ([]string, bool) {
		options, found := c.ListRegistryOptions(name)
		names := make([]string, 0, len(options))
		for _, one := range options {
			names = append(names, one.Value)
		}
		return names, found
	}
}

// ListRegistryOptions returns the options for the named registry when known.
func (c Config) ListRegistryOptions(name string) (manifest.SettingOptions, bool) {
	if name != settingsadapter.RegistryThemes {
		return nil, false
	}
	files, err := c.openThemeFiles()
	if err != nil {
		c.handleError(err)
	}
	var options manifest.SettingOptions
	for _, one := range themeusecase.NewListThemes(c.getThemeSources(files)).Execute() {
		options = append(options, manifest.SettingOption{Value: one.Name, Title: one.Title})
	}
	return options, true
}

// ListAppSettings returns the application settings declared.
func (c Config) ListAppSettings() []plugindomain.Setting {
	return settingsusecase.NewListAppSettings(
		settingsadapter.JSONCodec{}, c.store(), settingsadapter.Defaults{}, settingsadapter.AppDeclaration{}, c,
	).Execute()
}

// SubscribeSettingChanges starts listening to the settings that change. See settingsport.ChangeSubscriber.
func (c Config) SubscribeSettingChanges() (<-chan []settingsdomain.Setting, func()) {
	return c.feed().Subscribe()
}

// RefreshSettingEvents notifies setting subscribers of any change on disk or in memory.
func (c Config) RefreshSettingEvents() {
	c.feed().Refresh()
}

// SyncSetting returns whether note title and filename synchronization is enabled.
func (c Config) SyncSetting() query.SyncSetting {
	return func() query.SyncTitleAndFilename {
		target, err := c.Settings()
		if err != nil {
			c.handleError(fmt.Errorf("read settings: %w", err))
		}
		return query.SyncTitleAndFilename(target.Sync())
	}
}

// ReadSettings returns a function that reads settings as JSON and returns the settings file path.
func (c Config) ReadSettings() func() (string, string, error) {
	return c.store().ReadJSON
}

// GetSettingsWriter patches specific settings paths in the configuration file. The readers of the
// setting-changed event receive the settings whose value changed.
func (c Config) GetSettingsWriter() func(entries []settingsdomain.Setting) error {
	return c.writeSettings().Execute
}

// SettingsFile returns the path to the active settings file.
func (c Config) SettingsFile() (string, error) {
	if c.SettingsPath != "" {
		return c.SettingsPath, nil
	}
	if file, selected := c.GetAdjacentPath("numen.json"); selected {
		return file, nil
	}
	return settingsadapter.Path()
}

// GetAdjacentPath returns a path adjacent to RegistryPath when RegistryPath is set.
func (c Config) GetAdjacentPath(name string) (string, bool) {
	if c.RegistryPath == "" {
		return "", false
	}
	return filepath.Join(filepath.Dir(c.RegistryPath), name), true
}

// OpenGetSettings builds the settings reader use case for the given agent provider.
func (c Config) OpenGetSettings(provider agentport.AgentProvider) *settingsusecase.Get {
	return settingsusecase.NewGet(c.ReadSettings(), c.ModelsForProvider(provider), c.Plugins(), c.ListAppSettings)
}

// OpenWriteSettings builds the settings writer use case.
func (c Config) OpenWriteSettings() *settingsusecase.WriteSettings {
	return c.writeSettings()
}

// OpenWatchSettings builds the settings change watcher use case.
func (c Config) OpenWatchSettings() *settingsusecase.WatchSettingChanges {
	return settingsusecase.NewWatchSettingChanges(c.SubscribeSettingChanges)
}

// OpenChoosePluginPath builds the plugin path picker use case.
func (c Config) OpenChoosePluginPath(dialog machineport.PathDialog) *settingsusecase.ChoosePluginPath {
	return settingsusecase.NewChoosePluginPath(dialog, c.Plugins(), c.GetSettingsWriter())
}

// OpenRetryPlugin builds the plugin retry use case.
func (c Config) OpenRetryPlugin(retrier func(context.Context, string) (*plugindomain.Failure, error)) *settingsusecase.RetryPlugin {
	retry, err := settingsusecase.NewRetryPlugin(retrier)
	if err != nil {
		return nil
	}
	return retry
}

func (c Config) store() *settingsadapter.Store {
	var secrets settingsadapter.SecretDeclarations
	if c.PluginHost != nil {
		secrets = c.PluginHost
	}
	return settingsadapter.NewStore(c.SettingsFiles, c.SettingsFile, c.GetDeclarations(), c.GetRegistries(), secrets)
}

func (c Config) feed() settingevents.Feed {
	list := c.listSettingValues()
	return settingevents.Feed{
		Events:  c.SettingEvents,
		GetPath: c.SettingsFile,
		Read:    list.Execute,
		Initial: func() settingsdomain.Values {
			values, err := list.ListDefaults()
			if err != nil {
				c.handleError(err)
			}
			return values
		},
		Report: c.handleError,
		Hub:    c.Watches,
	}
}

func (c Config) listSettingValues() *settingsusecase.ListSettingValues {
	var plugins settingsport.PluginInfos
	if c.PluginHost != nil {
		plugins = c.PluginHost
	}
	var late settingsport.LatePrograms
	if c.Programs != nil {
		late = c.Programs
	}
	return settingsusecase.NewListSettingValues(
		settingsadapter.JSONCodec{}, c.store(), settingsadapter.Defaults{}, settingsadapter.AppDeclaration{}, plugins, late,
	)
}

func (c Config) writeSettings() *settingsusecase.WriteSettings {
	var plugins settingsport.PluginSettings
	if c.PluginHost != nil {
		plugins = c.PluginHost
	}
	check := settingsusecase.NewCheckSettings(
		settingsadapter.JSONCodec{}, settingsadapter.NewValidator(c.GetDeclarations(), c.GetRegistries()), plugins,
	)
	return settingsusecase.NewWriteSettings(check, c.store(), c.feed())
}

func (c Config) notifyPlugins() *settingsusecase.NotifyPlugins {
	var signal func()
	if c.PluginChanges != nil {
		signal = c.PluginChanges.Signal
	}
	return settingsusecase.NewNotifyPlugins(
		settingsadapter.JSONCodec{}, c.PluginHost, c.PluginHost, c.refreshPluginSecrets, c.resolveProfile, signal, c.handleError,
	)
}

func (c Config) refreshPluginSecrets(ctx context.Context) error {
	if err := c.SecretGate.Enter(ctx); err != nil {
		return err
	}
	defer c.SecretGate.Leave()
	return c.PluginHost.Refresh(ctx)
}

func (c Config) resolveProfile(profile string) string {
	return hardware.ResolveCompute(profile, c.Hardware).Profile
}

func (c Config) handleError(err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
}
