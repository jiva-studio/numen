package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
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
	indexingAt  = "indexing"
	embeddingAt = "embedding"
	agentAt     = "agent"
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
	return c.getSettingsAt(path)
}

// getSettingsAt is that document read from an explicit file.
func (c Config) getSettingsAt(path string) (Settings, error) {
	target := DefaultSettings()
	if err := settings.OpenAt(path, &target); err != nil {
		return Settings{}, err
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
		if target, err := c.getSettingsAt(path); err == nil {
			return target
		}
	}
	return DefaultSettings()
}

// SetSettings returns the configuration with the values of a settings file
// applied. Both entry points use it. An empty embedding model disables
// embedding.
func (c Config) SetSettings(s Settings) Config {
	c.Embedding = s.Indexing.Embedding
	c.Agent = s.Agent
	c.Performance = s.Performance
	c.Hardware = hardware.Detect()
	c.ResolvedCompute = hardware.ResolveCompute(s.Performance.Profile, c.Hardware)
	if c.PluginHost != nil {
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
					Path: []string{pluginSettingsPrefix, pluginID, key},
					JSON: string(raw),
				},
			})
		})
		c.PluginHost.SetKeyStore(c.KeyStore)
		c.setCompleter()
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
	return settings.AgentModels(agentAt, target.Agent, c.getAgentPrograms(c.agentProvider, target.Agent))
}

func (c Config) modelsForProvider(provider agentport.AgentProvider) func() []settingsport.Model {
	c.agentProvider = provider
	return c.listModels
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
			if len(setting.Replaces) == 0 || hasKeys(file, "plugins", plugin.ID, setting.Key) {
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
			Path: []string{pluginSettingsPrefix, pluginID, setting.Key}, JSON: string(content),
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
