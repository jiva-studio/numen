package container

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// MigratePluginSettings gives a person's plugin the settings that were theirs before the plugin took
// them over. A setting whose manifest says what it replaces is written into the plugin's section of
// the file from the first place the file still holds the old one, unless the section already holds
// the setting. The old keys stay in the file as the person wrote them.
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
	if json.Unmarshal(raw, &file) != nil {
		return nil //nolint:nilerr // corrupt settings file is skipped
	}

	moved := collectPluginSettingsMigrations(c.PluginHost.GetPluginInfos(), file)
	if len(moved) == 0 {
		return nil
	}
	return c.GetSettingsWriter()(moved)
}

func collectPluginSettingsMigrations(plugins []port.PluginInfo, file map[string]any) []port.Setting {
	var moved []port.Setting
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

func migrateOneSetting(pluginID string, setting port.PluginSetting, file map[string]any) (port.Setting, bool) {
	for _, place := range setting.Replaces {
		value, found := findSettingsKeys(file, strings.Split(place, ".")...)
		if !found {
			continue
		}
		content, err := json.Marshal(value)
		if err != nil {
			continue
		}
		return port.Setting{
			Path: []string{pluginSettingsPrefix, pluginID, setting.Key}, JSON: string(content),
		}, true
	}
	return port.Setting{}, false
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
