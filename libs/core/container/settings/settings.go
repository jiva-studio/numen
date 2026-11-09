// Package settings wires the settings file, setting change events and themes.
package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"

	settingsadapter "github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/pluginchanges"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
	settingsusecase "github.com/jiva-studio/numen/libs/core/usecase/settings"
)

// AgentAt is the key of the agent section in the settings file.
const AgentAt = "agent"

// Settings is the file that configures this installation: one object whose
// fields are its sections, in file order. The settings adapter owns the file
// and the window sections; each adapter defines the shape of its own section,
// and the union of the sections is assembled here, where every adapter is
// bound.
type Settings = settingsadapter.File

// Config wires the settings, setting change events and themes of an installation.
type Config struct {
	SettingsPath              string
	ThemesPath                string
	RegistryPath              string
	InterfaceScale, TextScale float64
	Hardware                  hardware.Specs
	ResolvedCompute           hardware.ResolvedCompute
	PluginHost                pluginport.PluginManager
	KeyStore                  pluginport.KeyStore
	Watches                   *vaultwatch.Hub
	SettingsFiles             *settingsadapter.Files
	SettingEvents             *settingevents.Events
	SecretGate                *secret.Gate
	SubscribedHosts           *sync.Map
	PluginChanges             *pluginchanges.Signals
	ErrorHandler              func(error)
	AgentProvider             func() agentport.AgentProvider
	GetAgentPrograms          func(agentport.AgentProvider, settingsadapter.Agent) []agentport.AgentProgram
	SetSecretSource           func()
}

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() Settings {
	return settingsadapter.DefaultFile()
}

// Settings reads the settings file. An unconfigured installation returns the
// default settings.
func (c Config) Settings() (Settings, error) {
	path, err := c.SettingsFile()
	if err != nil {
		return Settings{}, err
	}
	return c.GetCheckedSettingsAt(path)
}

// GetSettingsAt reads the settings file at path without filtering plugin declarations.
func (c Config) GetSettingsAt(path string) (Settings, error) {
	target := DefaultSettings()
	if err := c.SettingsFiles.OpenAt(path, &target); err != nil {
		return Settings{}, err
	}
	return target, nil
}

// ReadStoredSettings reads the settings file as it is stored, without leaving anything out of force.
// An unconfigured installation returns the default settings.
func (c Config) ReadStoredSettings() (Settings, error) {
	path, err := c.SettingsFile()
	if err != nil {
		return Settings{}, err
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return DefaultSettings(), nil
	}
	return c.GetSettingsAt(path)
}

// GetCheckedSettingsAt reads the settings file at path and drops invalid values.
func (c Config) GetCheckedSettingsAt(path string) (Settings, error) {
	target, err := c.GetSettingsAt(path)
	if err != nil {
		return Settings{}, err
	}
	if c.PluginHost != nil {
		target.DropInvalidConnections()
		target.DropInvalid(c.GetDeclarations(), c.GetRegistries())
		target.DropInvalidSecrets(c.IsSecretDeclared)
	} else {
		target.DropInvalidSecrets(func(pluginport.DeclarationKey) bool { return true })
		target.DropInvalidConnections()
	}
	return target, nil
}

// GetSettingsOrDefaults returns the checked settings or DefaultSettings on failure.
func (c Config) GetSettingsOrDefaults() Settings {
	if path, err := c.SettingsFile(); err == nil {
		if target, err := c.GetCheckedSettingsAt(path); err == nil {
			return target
		}
	}
	return DefaultSettings()
}

// ResolveCompute detects the machine hardware and resolves the compute plan for s.
func (c Config) ResolveCompute(s Settings) (hardware.Specs, hardware.ResolvedCompute) {
	specs := hardware.Detect()
	return specs, hardware.ResolveCompute(s.Performance.Profile, specs)
}

// ConfigurePluginHost applies the settings s to the plugin host and subscribes it to changes.
func (c Config) ConfigurePluginHost(s Settings) {
	if c.PluginHost == nil {
		return
	}
	c.PluginHost.SetPerformanceProfile(context.Background(), c.ResolvedCompute.Profile)
	if s.Plugins != nil {
		c.PluginHost.SetSettings(s.Plugins)
	}
	c.PluginHost.SetPersistHandler(func(pluginID string, key string, value any) error {
		writer := c.GetSettingsWriter()
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return writer([]settingsport.Setting{
			{
				Path: []string{pluginID, key},
				JSON: string(raw),
			},
		})
	})
	c.PluginHost.SetKeyStore(c.KeyStore)
	if c.SetSecretSource != nil {
		c.SetSecretSource()
	}
	c.SubscribeHost()
}

// Plugins returns a function that lists the discovered plugins and their
// settings schemas.
func (c Config) Plugins() func() []pluginport.PluginInfo {
	return func() []pluginport.PluginInfo {
		if c.PluginHost == nil {
			return nil
		}
		var discoverer pluginport.PluginDiscoverer = c.PluginHost
		return discoverer.GetPluginInfos()
	}
}

// Models returns a function that lists the models each model setting accepts,
// and the programs the agent setting accepts.
func (c Config) Models() func() []settingsport.Model {
	var provider agentport.AgentProvider
	if c.AgentProvider != nil {
		provider = c.AgentProvider()
	}
	return c.ModelsForProvider(provider)
}

// ModelsForProvider returns a function that lists the models for the given agent provider.
func (c Config) ModelsForProvider(provider agentport.AgentProvider) func() []settingsport.Model {
	return func() []settingsport.Model {
		target := c.GetSettingsOrDefaults()
		var programs []agentport.AgentProgram
		if c.GetAgentPrograms != nil {
			programs = c.GetAgentPrograms(provider, target.Agent)
		}
		return PrefixModels(settingsadapter.OwnerApp, settingsadapter.AgentModels(AgentAt, target.Agent, programs))
	}
}

// PrefixModels prepends owner to the path of each model and its writes.
func PrefixModels(owner string, models []settingsport.Model) []settingsport.Model {
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
	return func(owner string) ([]pluginport.PluginManifestSetting, bool) {
		if owner == settingsadapter.OwnerApp {
			return settingsadapter.GetAppDeclaration(), true
		}
		if c.PluginHost == nil {
			return nil, false
		}
		var declarer pluginport.PluginSettingDeclarer = c.PluginHost
		return declarer.GetPluginSettingDeclarations(owner)
	}
}

// GetRegistries returns the option values for a named setting registry.
func (c Config) GetRegistries() settingsadapter.Registries {
	return func(name string) ([]string, bool) {
		if name == settingsadapter.RegistryConnections {
			return c.listConnectionIDs()
		}
		options, found := c.ListRegistryOptions(name)
		names := make([]string, 0, len(options))
		for _, one := range options {
			names = append(names, one.Value)
		}
		return names, found
	}
}

func (c Config) listConnectionIDs() ([]string, bool) {
	stored, err := c.ReadStoredSettings()
	if err != nil {
		c.handleError(err)
		return nil, false
	}
	stored.DropInvalidConnections()
	ids := make([]string, 0, len(stored.Connections))
	for _, one := range stored.Connections {
		ids = append(ids, one.ID)
	}
	return ids, true
}

// ListRegistryOptions returns the options for the named registry when known.
func (c Config) ListRegistryOptions(name string) (pluginport.PluginSettingOptions, bool) {
	if name != settingsadapter.RegistryThemes {
		return nil, false
	}
	catalog, err := c.getCatalogue()
	if err != nil {
		c.handleError(err)
	}
	var options pluginport.PluginSettingOptions
	for _, one := range catalog.Themes() {
		options = append(options, pluginport.PluginSettingOption{Value: one.Name, Title: one.Title})
	}
	return options, true
}

// ListAppSettings returns the application settings declared.
func (c Config) ListAppSettings() []pluginport.PluginSetting {
	current := c.GetSettingsOrDefaults()
	declared := settingsadapter.GetAppDeclaration()
	entries := make([]pluginport.PluginSetting, 0, len(declared))
	for _, one := range declared {
		options := one.Options
		if one.OptionsFrom != "" {
			options, _ = c.ListRegistryOptions(one.OptionsFrom)
		}
		defaultJSON := ""
		if encoded, err := json.Marshal(one.Default); err == nil && one.Default != nil {
			defaultJSON = string(encoded)
		}
		setting := pluginport.NewPluginSetting(one, defaultJSON, options)
		setting.ActiveOptions = options
		setting.IsShown = true
		if value, found := current.GetValue(append([]string{settingsadapter.OwnerApp}, strings.Split(one.Key, ".")...)...); found {
			if encoded, err := json.Marshal(value); err == nil {
				setting.ValueJSON = string(encoded)
			}
		}
		entries = append(entries, setting)
	}
	return entries
}

// SubscribeSettingChanges starts listening to the settings that change. See settingsport.ChangeSubscriber.
func (c Config) SubscribeSettingChanges() (<-chan []settingsport.Setting, func()) {
	if c.SettingEvents == nil {
		closed := make(chan []settingsport.Setting)
		close(closed)
		return closed, func() {}
	}
	path, err := c.SettingsFile()
	if err != nil {
		c.handleError(err)
		closed := make(chan []settingsport.Setting)
		close(closed)
		return closed, func() {}
	}
	return c.SettingEvents.Subscribe(settingevents.Source{
		Read:    c.readSettingValues,
		Initial: c.listDefaultSettingValues,
		Report:  c.handleError,
		Hub:     c.Watches,
		Path:    path,
	})
}

// RefreshSettingEvents notifies setting subscribers of any change on disk or in memory.
func (c Config) RefreshSettingEvents() {
	if c.SettingEvents != nil {
		c.SettingEvents.Refresh(c.readSettingValues)
	}
}

func (c Config) readSettingValues() (settingevents.Values, error) {
	target, err := c.ReadCheckedSettings()
	if err != nil {
		return nil, err
	}
	return c.listSettingValues(target)
}

// ReadCheckedSettings reads and validates the active settings file, or returns defaults if missing.
func (c Config) ReadCheckedSettings() (Settings, error) {
	path, err := c.SettingsFile()
	if err != nil {
		return Settings{}, err
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return DefaultSettings(), nil
	}
	return c.GetCheckedSettingsAt(path)
}

func (c Config) listDefaultSettingValues() settingevents.Values {
	values, err := c.listSettingValues(DefaultSettings())
	if err != nil {
		c.handleError(err)
	}
	return values
}

// hostFieldEnabled is the field of the object of a plugin that the host keeps. A plugin that has no
// stored value for it is enabled.
const hostFieldEnabled = "enabled"

const settingKeySecrets = "secrets"

const settingKeyConnections = "connections"

func (c Config) listSettingValues(target Settings) (settingevents.Values, error) {
	values := settingevents.Values{}
	var failures []error
	add := func(path []string, value any) {
		encoded, err := json.Marshal(value)
		if err != nil {
			failures = append(failures, err)
			return
		}
		values[settingevents.JoinPath(path)] = string(encoded)
	}
	for _, declared := range settingsadapter.GetAppDeclaration() {
		path := append([]string{settingsadapter.OwnerApp}, strings.Split(declared.Key, ".")...)
		if value, found := target.GetValue(path...); found {
			add(path, value)
		}
	}
	for _, plugin := range c.Plugins()() {
		listPluginValues(target, plugin, add, values)
	}
	if len(target.Secrets) > 0 {
		content, err := target.WriteSecretsJSON()
		if err != nil {
			failures = append(failures, err)
		} else {
			values[settingevents.JoinPath([]string{settingKeySecrets})] = content
		}
	}
	if len(target.Connections) > 0 {
		content, err := target.WriteConnectionsJSON()
		if err != nil {
			failures = append(failures, err)
		} else {
			values[settingevents.JoinPath([]string{settingKeyConnections})] = content
		}
	}
	return values, errors.Join(failures...)
}

func listPluginValues(target Settings, plugin pluginport.PluginInfo, add func([]string, any), values settingevents.Values) {
	for _, declared := range plugin.Settings {
		path := []string{plugin.ID, declared.Key}
		if value, found := target.GetValue(path...); found {
			add(path, value)
		} else if declared.DefaultJSON != "" {
			values[settingevents.JoinPath(path)] = declared.DefaultJSON
		}
	}
	enabled := true
	if value, found := target.GetValue(plugin.ID, hostFieldEnabled); found {
		if stored, isBool := value.(bool); isBool {
			enabled = stored
		}
	}
	add([]string{plugin.ID, hostFieldEnabled}, enabled)
}

// hostSubscription is the delivery of setting changes to one plugin host.
type hostSubscription struct {
	mu          sync.Mutex
	unsubscribe func()
	ended       chan struct{}
}

// SubscribeHost subscribes the plugin host to setting change events once per host.
func (c Config) SubscribeHost() {
	if c.SettingEvents == nil || c.PluginHost == nil {
		return
	}
	var one *hostSubscription
	if c.SubscribedHosts != nil && reflect.TypeOf(c.PluginHost).Comparable() {
		one = &hostSubscription{}
		one.mu.Lock()
		defer one.mu.Unlock()
		if _, isSubscribed := c.SubscribedHosts.LoadOrStore(c.PluginHost, one); isSubscribed {
			return
		}
	}
	changes, unsubscribe := c.SubscribeSettingChanges()
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		for batch := range changes {
			c.NotifyHost(batch)
		}
	}()
	if one != nil {
		one.unsubscribe, one.ended = unsubscribe, ended
	}
}

// UnsubscribeHost ends the delivery of setting changes to the plugin host and returns after the
// batch being delivered has been handled.
func (c Config) UnsubscribeHost() {
	if c.SubscribedHosts == nil || c.PluginHost == nil || !reflect.TypeOf(c.PluginHost).Comparable() {
		return
	}
	found, isSubscribed := c.SubscribedHosts.LoadAndDelete(c.PluginHost)
	if !isSubscribed {
		return
	}
	one, isHostSubscription := found.(*hostSubscription)
	if !isHostSubscription {
		return
	}
	one.mu.Lock()
	defer one.mu.Unlock()
	one.unsubscribe()
	<-one.ended
}

// RefreshPluginSecrets refreshes the plugin host under the secret gate.
func (c Config) RefreshPluginSecrets(ctx context.Context) error {
	if err := c.SecretGate.Enter(ctx); err != nil {
		return err
	}
	defer c.SecretGate.Leave()
	return c.PluginHost.Refresh(ctx)
}

// NotifyHost forwards a batch of changed settings to the plugin host.
func (c Config) NotifyHost(batch []settingsport.Setting) {
	NotifyPluginSettings(c.PluginHost, batch, c.handleError)
	if c.PluginChanges != nil {
		defer c.PluginChanges.Signal()
	}
	for _, one := range batch {
		if slices.Equal(one.Path, []string{settingKeySecrets}) {
			if err := c.RefreshPluginSecrets(context.Background()); err != nil {
				c.handleError(fmt.Errorf("configure the plugins after the secrets changed: %w", err))
			}
			continue
		}
		if !slices.Equal(one.Path, []string{settingsadapter.OwnerApp, "performance", "profile"}) {
			continue
		}
		var profile string
		if err := json.Unmarshal([]byte(one.JSON), &profile); err != nil {
			c.handleError(err)
			continue
		}
		c.PluginHost.SetPerformanceProfile(context.Background(), hardware.ResolveCompute(profile, c.Hardware).Profile)
	}
}

// IsSecretDeclared reports whether the plugin host declares permission.
func (c Config) IsSecretDeclared(permission pluginport.DeclarationKey) bool {
	return slices.Contains(c.PluginHost.ListDeclarations(), permission)
}

// SyncSetting returns whether note title and filename synchronization is enabled.
func (c Config) SyncSetting() note.SyncSetting {
	return func() note.SyncTitleAndFilename {
		target, err := c.Settings()
		if err != nil {
			c.handleError(fmt.Errorf("read settings: %w", err))
		}
		return note.SyncTitleAndFilename(target.Sync())
	}
}

// ReadSettings returns a function that reads settings as JSON and returns the settings file path.
func (c Config) ReadSettings() func() (string, string, error) {
	return func() (string, string, error) {
		path, err := c.SettingsFile()
		if err != nil {
			return "", "", err
		}
		target, err := c.GetCheckedSettingsAt(path)
		if err != nil {
			return "", path, err
		}
		content, err := target.WriteJSON()
		return content, path, err
	}
}

// GetSettingsWriter patches specific settings paths in the configuration file. The readers of the
// setting-changed event receive the settings whose value changed.
func (c Config) GetSettingsWriter() func(entries []settingsport.Setting) error {
	return c.WriteSettings
}

// WriteSettings validates and writes entries to the settings file.
func (c Config) WriteSettings(entries []settingsport.Setting) error {
	target := make([]settingsadapter.Setting, 0, len(entries))
	for _, one := range entries {
		var value json.RawMessage
		if err := json.Unmarshal([]byte(one.JSON), &value); err != nil {
			return fmt.Errorf("%w: %w", settingsport.ErrNotASetting, err)
		}
		target = append(target, settingsadapter.Setting{At: one.Path, Written: value})
	}
	if err := c.CheckWrites(entries); err != nil {
		return fmt.Errorf("%w: %w", settingsport.ErrNotASetting, err)
	}
	path, err := c.SettingsFile()
	if err != nil {
		return err
	}
	into := DefaultSettings()
	if err := c.SettingsFiles.Save(path, &into, target...); err != nil {
		return err
	}
	c.RefreshSettingEvents()
	return nil
}

// CheckWrites validates entries against the declared settings and registries.
func (c Config) CheckWrites(entries []settingsport.Setting) error {
	declarations := c.GetDeclarations()
	registries := c.GetRegistries()
	for _, one := range entries {
		var val any
		if err := json.Unmarshal([]byte(one.JSON), &val); err != nil {
			return err
		}
		if err := settingsadapter.CheckWrite(declarations, registries, one.Path, val); err != nil {
			return err
		}
		if c.PluginHost != nil && IsPluginSetting(one.Path) {
			if err := c.PluginHost.CheckPluginSetting(one.Path[0], one.Path[1], val); err != nil {
				return err
			}
		}
	}
	return nil
}

// IsPluginSetting reports whether path names a plugin setting.
func IsPluginSetting(path []string) bool {
	return len(path) == 2 && path[0] != settingsadapter.OwnerApp
}

// NotifyPluginSettings updates the plugin host for each plugin setting in entries.
func NotifyPluginSettings(host pluginport.PluginConfigurator, entries []settingsport.Setting, report func(error)) {
	if host == nil {
		return
	}
	// The enabled entries go last: enabling a plugin generates the values of its settings that hold
	// none, and a value in the same batch is in memory by then.
	isEnabledEntry := func(one settingsport.Setting) bool {
		return len(one.Path) == 2 && one.Path[1] == hostFieldEnabled
	}
	ordered := slices.Concat(
		slices.DeleteFunc(slices.Clone(entries), isEnabledEntry),
		slices.DeleteFunc(slices.Clone(entries), func(one settingsport.Setting) bool { return !isEnabledEntry(one) }),
	)
	for _, one := range ordered {
		if !IsPluginSetting(one.Path) {
			continue
		}
		var val any
		if err := json.Unmarshal([]byte(one.JSON), &val); err == nil {
			if err := host.UpdatePluginSetting(context.Background(), one.Path[0], one.Path[1], val); err != nil {
				report(fmt.Errorf("update plugin setting %s: %w", strings.Join(one.Path, "."), err))
			}
		}
	}
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
func (c Config) OpenWriteSettings() *settingsusecase.Write {
	return settingsusecase.NewWrite(c.GetSettingsWriter())
}

// OpenWatchSettings builds the settings change watcher use case.
func (c Config) OpenWatchSettings() *settingsusecase.Watch {
	return settingsusecase.NewWatch(c.SubscribeSettingChanges)
}

// OpenChoosePluginPath builds the plugin path picker use case.
func (c Config) OpenChoosePluginPath(dialog machineport.PathDialog) *settingsusecase.ChoosePluginPath {
	return settingsusecase.NewChoosePluginPath(dialog, c.Plugins(), c.GetSettingsWriter())
}

// OpenRetryPlugin builds the plugin retry use case.
func (c Config) OpenRetryPlugin(retrier func(context.Context, string) (*pluginport.PluginFailure, error)) *settingsusecase.RetryPlugin {
	retry, err := settingsusecase.NewRetryPlugin(retrier)
	if err != nil {
		return nil
	}
	return retry
}

func (c Config) handleError(err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
}
