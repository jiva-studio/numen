package container

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/usecase/trigger"
)

// openTriggers is the triggers of the enabled plugins over the files of a vault, which read the
// settings as get reads them; nil where the installation runs no plugins.
func (c Config) openTriggers(get *settings.Get) *trigger.Triggers {
	if c.PluginHost == nil || c.SafeMode || get == nil {
		return nil
	}
	return trigger.NewTriggers(
		commandbridge.Open(c.PluginHost), c.VaultReaders(), c.PluginHost,
		derivedfile.TriggerRecords{Options: c.vaultOptions(), Folder: coreFolder},
		func() (map[string]any, error) {
			current, err := get.Execute()
			if err != nil {
				return nil, err
			}
			return createTriggerConfig(current.Written, current.App, current.Plugins)
		},
	)
}

// configAppOwner is the owner of the settings of the application in the settings file and in config.
const configAppOwner = "app"

// createTriggerConfig is the settings as when conditions read them under config, built as the window builds
// it: each setting at config.app.<key> or config.<plugin id split at dots>.<key>, the key split at
// dots, holding the value the settings file holds for its owner and key, or its declared default.
// A declared default that is no JSON leaves the setting out.
func createTriggerConfig(settingsFile string, app []pluginport.PluginSetting, plugins []pluginport.PluginInfo) (map[string]any, error) {
	var file map[string]map[string]any
	if err := json.Unmarshal([]byte(settingsFile), &file); err != nil {
		return nil, fmt.Errorf("read the settings file: %w", err)
	}
	config := map[string]any{}
	add := func(owner string, path []string, settings []pluginport.PluginSetting) {
		for _, one := range settings {
			value, isWritten := file[owner][one.Key]
			if !isWritten && json.Unmarshal([]byte(one.DefaultJSON), &value) != nil {
				continue
			}
			setConfigAt(config, append(append([]string(nil), path...), strings.Split(one.Key, ".")...), value)
		}
	}
	add(configAppOwner, []string{configAppOwner}, app)
	for _, plugin := range plugins {
		add(plugin.ID, strings.Split(plugin.ID, "."), plugin.Settings)
	}
	return config, nil
}

// setConfigAt puts the value at the path of nested maps, making the maps on the way.
func setConfigAt(config map[string]any, path []string, value any) {
	level := config
	for _, part := range path[:len(path)-1] {
		next, ok := level[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			level[part] = next
		}
		level = next
	}
	level[path[len(path)-1]] = value
}
