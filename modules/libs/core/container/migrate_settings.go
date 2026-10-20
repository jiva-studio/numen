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
	if err := json.Unmarshal(raw, &file); err != nil {
		// A file that is not JSON is read, and refused, where the settings are read.
		return nil
	}

	var moved []port.Setting
	for _, plugin := range c.PluginHost.GetPluginInfos() {
		for _, setting := range plugin.Settings {
			if len(setting.Replaces) == 0 || hasKeys(file, "plugins", plugin.ID, setting.Key) {
				continue
			}
			for _, place := range setting.Replaces {
				value, found := lookKeys(file, strings.Split(place, ".")...)
				if !found {
					continue
				}
				written, err := json.Marshal(value)
				if err != nil {
					continue
				}
				moved = append(moved, port.Setting{
					Path: []string{"plugins", plugin.ID, setting.Key}, JSON: string(written),
				})
				break
			}
		}
	}
	if len(moved) == 0 {
		return nil
	}
	return c.GetSettingsWriter()(moved)
}

// lookKeys is the value at the keys of a decoded document, and whether the document holds one.
func lookKeys(document map[string]any, keys ...string) (any, bool) {
	var at any = document
	for _, key := range keys {
		object, isObject := at.(map[string]any)
		if !isObject {
			return nil, false
		}
		next, found := object[key]
		if !found {
			return nil, false
		}
		at = next
	}
	return at, true
}

func hasKeys(document map[string]any, keys ...string) bool {
	_, found := lookKeys(document, keys...)
	return found
}
