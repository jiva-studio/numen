package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/settingevents"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// The settings adapter owns the file and the sections about the window; every
// other section belongs to the adapter it configures, and is put beside them
// here, where every adapter is bound.
const (
	agentAt = "agent"
)

// Settings is the file that configures this installation: one object whose
// fields are its sections, in file order. The settings adapter owns the file
// and the window sections; each adapter defines the shape of its own section,
// and the union of the sections is assembled here, where every adapter is
// bound.
type Settings = settings.File

// Settings reads the settings file. An unconfigured installation returns the
// default settings.
func (c Config) Settings() (Settings, error) {
	path, err := c.settingsFile()
	if err != nil {
		return Settings{}, err
	}
	return c.getCheckedSettingsAt(path)
}

func (c Config) getSettingsAt(path string) (Settings, error) {
	target := DefaultSettings()
	if err := c.getResources().settingsFiles.OpenAt(path, &target); err != nil {
		return Settings{}, err
	}
	return target, nil
}

func (c Config) getCheckedSettingsAt(path string) (Settings, error) {
	target, err := c.getSettingsAt(path)
	if err != nil {
		return Settings{}, err
	}
	if c.PluginHost != nil {
		target.DropInvalid(c.getDeclarations(), c.getRegistries())
		target.DropInvalidSecrets(c.isSecretDeclared)
	} else {
		target.DropInvalidSecrets(func(pluginport.SecretPermission) bool { return true })
	}
	return target, nil
}

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() Settings {
	return settings.DefaultFile()
}

func (c Config) getSettingsOrDefaults() Settings {
	if path, err := c.settingsFile(); err == nil {
		if target, err := c.getCheckedSettingsAt(path); err == nil {
			return target
		}
	}
	return DefaultSettings()
}

// SetSettings returns the configuration with the values of a settings file
// applied. Both entry points use it.
func (c Config) SetSettings(s Settings) Config {
	c.Hardware = hardware.Detect()
	c.ResolvedCompute = hardware.ResolveCompute(s.Performance.Profile, c.Hardware)
	if c.PluginHost != nil {
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
		c.setSecretSource()
		c.setCompleter()
		c.subscribeHost()
	}
	return c
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
//
// The settings are read with them, so every row reflects the effective
// settings; an unreadable file uses the defaults. The programs and their
// models are the ones listed by the agent plugins.
func (c Config) Models() func() []settingsport.Model {
	return c.modelsForProvider(c.AgentProvider())
}

func (c Config) listModels() []settingsport.Model {
	target := c.getSettingsOrDefaults()
	return prefixModels(settings.OwnerApp, settings.AgentModels(agentAt, target.Agent, c.getAgentPrograms(c.agentProvider, target.Agent)))
}

func prefixModels(owner string, models []settingsport.Model) []settingsport.Model {
	for place := range models {
		models[place].Path = append([]string{owner}, models[place].Path...)
		for index := range models[place].Writes {
			models[place].Writes[index].Path = append([]string{owner}, models[place].Writes[index].Path...)
		}
	}
	return models
}

func (c Config) modelsForProvider(provider agentport.AgentProvider) func() []settingsport.Model {
	c.agentProvider = provider
	c.programs = &settings.ProgramList{}
	return c.listModels
}

func (c Config) getDeclarations() settings.Declarations {
	return func(owner string) ([]pluginport.PluginManifestSetting, bool) {
		if owner == settings.OwnerApp {
			return settings.GetAppDeclaration(), true
		}
		if c.PluginHost == nil {
			return nil, false
		}
		var declarer pluginport.PluginSettingDeclarer = c.PluginHost
		return declarer.GetPluginSettingDeclarations(owner)
	}
}

func (c Config) getRegistries() settings.Registries {
	return func(name string) ([]string, bool) {
		options, found := c.listRegistryOptions(name)
		names := make([]string, 0, len(options))
		for _, one := range options {
			names = append(names, one.Value)
		}
		return names, found
	}
}

func (c Config) listRegistryOptions(name string) (pluginport.PluginSettingOptions, bool) {
	if name != settings.RegistryThemes {
		return nil, false
	}
	catalogue, err := c.getCatalogue()
	if err != nil {
		c.handleError(err)
	}
	var options pluginport.PluginSettingOptions
	for _, one := range catalogue.Themes() {
		options = append(options, pluginport.PluginSettingOption{Value: one.Name, Title: one.Title})
	}
	return options, true
}

func (c Config) listAppSettings() []pluginport.PluginSetting {
	return c.listAppSettingsOn(runtime.GOOS)
}

func (c Config) listAppSettingsOn(goos string) []pluginport.PluginSetting {
	current := c.getSettingsOrDefaults()
	declared := settings.GetAppDeclaration()
	entries := make([]pluginport.PluginSetting, 0, len(declared))
	for _, one := range declared {
		if !settings.IsAvailableOn(one.Key, goos) {
			continue
		}
		options := one.Options
		if one.OptionsFrom != "" {
			options, _ = c.listRegistryOptions(one.OptionsFrom)
		}
		defaultJSON := ""
		if encoded, err := json.Marshal(one.Default); err == nil && one.Default != nil {
			defaultJSON = string(encoded)
		}
		setting := pluginport.NewPluginSetting(one, defaultJSON, options)
		setting.ActiveOptions = options
		setting.IsShown = true
		if value, found := current.GetValue(append([]string{settings.OwnerApp}, strings.Split(one.Key, ".")...)...); found {
			if encoded, err := json.Marshal(value); err == nil {
				setting.ValueJSON = string(encoded)
			}
		}
		entries = append(entries, setting)
	}
	return entries
}

// SubscribeSettingChanges starts listening to the settings that change, whether a window, a plugin
// or the palette writes them or a person edits the file by hand. See settingsport.ChangeSubscriber.
func (c Config) SubscribeSettingChanges() (<-chan []settingsport.Setting, func()) {
	events := c.getResources().settingEvents
	if events == nil {
		closed := make(chan []settingsport.Setting)
		close(closed)
		return closed, func() {}
	}
	path, err := c.settingsFile()
	if err != nil {
		c.handleError(err)
		closed := make(chan []settingsport.Setting)
		close(closed)
		return closed, func() {}
	}
	return events.Subscribe(settingevents.Source{
		Read:    c.readSettingValues,
		Initial: c.listDefaultSettingValues,
		Report:  c.handleError,
		Hub:     c.getResources().watches,
		Path:    path,
	})
}

func (c Config) refreshSettingEvents() {
	if events := c.getResources().settingEvents; events != nil {
		events.Refresh(c.readSettingValues)
	}
}

func (c Config) readSettingValues() (settingevents.Values, error) {
	target, err := c.readCheckedSettings()
	if err != nil {
		return nil, err
	}
	return c.listSettingValues(target)
}

func (c Config) readCheckedSettings() (Settings, error) {
	path, err := c.settingsFile()
	if err != nil {
		return Settings{}, err
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return DefaultSettings(), nil
	}
	return c.getCheckedSettingsAt(path)
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

func secretsPath() []string { return []string{"secrets"} }

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
	for _, declared := range settings.GetAppDeclaration() {
		path := append([]string{settings.OwnerApp}, strings.Split(declared.Key, ".")...)
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
			values[settingevents.JoinPath(secretsPath())] = content
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

func (c Config) subscribeHost() {
	resources := c.getResources()
	if resources.settingEvents == nil || c.PluginHost == nil {
		return
	}
	resources.hostSubscribed.Do(func() {
		changes, _ := c.SubscribeSettingChanges()
		go func() {
			for batch := range changes {
				c.tellHost(batch)
			}
		}()
	})
}

func (c Config) tellHost(batch []settingsport.Setting) {
	notifyPluginSettings(c.PluginHost, batch, c.handleError)
	for _, one := range batch {
		if slices.Equal(one.Path, secretsPath()) {
			if err := c.PluginHost.RefreshSecrets(context.Background()); err != nil {
				c.handleError(fmt.Errorf("configure the plugins after the secrets changed: %w", err))
			}
			continue
		}
		if !slices.Equal(one.Path, profilePath()) {
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

func profilePath() []string { return []string{settings.OwnerApp, "performance", "profile"} }

func (c Config) isSecretDeclared(permission pluginport.SecretPermission) bool {
	return slices.Contains(c.PluginHost.ListDeclarations(), permission)
}
