package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"strings"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"

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

// getSettingsAt is that document read from an explicit file.
func (c Config) getSettingsAt(path string) (Settings, error) {
	target := DefaultSettings()
	if err := c.getResources().settingsFiles.OpenAt(path, &target); err != nil {
		return Settings{}, err
	}
	return target, nil
}

// getCheckedSettingsAt is that document with every value left out of force that the declarations of
// the plugin host and the registries refuse. Without a plugin host no plugin declares anything,
// and the document is read as it is.
func (c Config) getCheckedSettingsAt(path string) (Settings, error) {
	target, err := c.getSettingsAt(path)
	if err != nil {
		return Settings{}, err
	}
	if c.PluginHost != nil {
		target.DropInvalid(c.getDeclarations(), c.getRegistries())
	}
	return target, nil
}

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() Settings {
	return settings.DefaultFile()
}

// getSettingsOrDefaults returns the settings document, or the defaults when
// the file cannot be read.
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

// prefixModels puts the owner at the head of the path of every model and of every setting it writes.
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

// getDeclarations returns the settings each owner declares: the shell, and the plugins of the host.
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

// getRegistries returns the registries an option list may name: the installed themes.
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

// listRegistryOptions returns the options a registry lists, and false for a registry that has no
// provider.
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

// listAppSettings returns the declaration of the settings of the shell with the options, the
// visibility and the value in force of each under the settings file.
func (c Config) listAppSettings() []pluginport.PluginSetting {
	current := c.getSettingsOrDefaults()
	declared := settings.GetAppDeclaration()
	entries := make([]pluginport.PluginSetting, 0, len(declared))
	for _, one := range declared {
		if !settings.IsAvailableOn(one.Key, runtime.GOOS) {
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

// MigratePluginSettings copies settings that existed before a plugin took
// them over into the plugin's section. A setting whose manifest names the key
// it replaces is written to the plugin's section of the file from the first
// location that still holds the old key, unless the section already holds the
// setting. The old keys remain in the file.
func (c Config) MigratePluginSettings() error {
	if c.PluginHost == nil {
		return nil
	}
	path, err := c.settingsFile()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var file map[string]any
	if err := json.Unmarshal(raw, &file); err != nil {
		c.handleError(fmt.Errorf("skip plugin settings migration, parse %s: %w", path, err))
		return nil
	}

	moved := collectPluginSettingsMigrations(c.PluginHost.GetPluginInfos(), file)
	if len(moved) == 0 {
		return nil
	}
	return c.GetSettingsWriter()(moved)
}

func collectPluginSettingsMigrations(plugins []pluginport.PluginInfo, file map[string]any) []settingsport.Setting {
	var moved []settingsport.Setting
	for _, plugin := range plugins {
		for _, setting := range plugin.Settings {
			if len(setting.Replaces) == 0 || hasKeys(file, plugin.ID, setting.Key) {
				continue
			}
			if s, ok := migrateOneSetting(plugin.ID, setting, file); ok {
				moved = append(moved, s)
			}
		}
	}
	return moved
}

func migrateOneSetting(pluginID string, setting pluginport.PluginSetting, file map[string]any) (settingsport.Setting, bool) {
	for _, place := range setting.Replaces {
		value, found := findSettingsKeys(file, strings.Split(place, ".")...)
		if !found {
			continue
		}
		content, err := json.Marshal(value)
		if err != nil {
			continue
		}
		return settingsport.Setting{
			Path: []string{pluginID, setting.Key}, JSON: string(content),
		}, true
	}
	return settingsport.Setting{}, false
}

// findSettingsKeys is the value at the keys of a decoded document, and whether the document holds one.
func findSettingsKeys(document map[string]any, keys ...string) (any, bool) {
	var pos any = document
	for _, key := range keys {
		object, isObject := pos.(map[string]any)
		if !isObject {
			return nil, false
		}
		next, found := object[key]
		if !found {
			return nil, false
		}
		pos = next
	}
	return pos, true
}

func hasKeys(document map[string]any, keys ...string) bool {
	_, found := findSettingsKeys(document, keys...)
	return found
}
