package pluginport

import (
	"encoding/json"
	"strings"
)

// WhenConfig gives the settings as the config variable of when conditions.
type WhenConfig interface {
	// GetWhenConfig returns the config, built by NewWhenConfig from the settings the host resolved.
	GetWhenConfig() map[string]any
}

// configAppOwner is the owner of the settings of the application in config.
const configAppOwner = "app"

// NewWhenConfig builds the config of when conditions from resolved settings: a setting of the
// application at config.app.<key>, and one of a plugin at config.<plugin id split at its dots>.<key>,
// the key split at its dots too. A setting holds its value, or its default where no value is set; a
// setting with neither is left out.
func NewWhenConfig(app []PluginSetting, plugins []PluginInfo) map[string]any {
	config := map[string]any{}
	addSettings(config, []string{configAppOwner}, app)
	for _, plugin := range plugins {
		addSettings(config, strings.Split(plugin.ID, "."), plugin.Settings)
	}
	return config
}

func addSettings(config map[string]any, owner []string, settings []PluginSetting) {
	for _, one := range settings {
		text := one.ValueJSON
		if text == "" {
			text = one.DefaultJSON
		}
		var value any
		if text == "" || json.Unmarshal([]byte(text), &value) != nil {
			continue
		}
		path := append(append([]string(nil), owner...), strings.Split(one.Key, ".")...)
		level := config
		for _, part := range path[:len(path)-1] {
			next, isMap := level[part].(map[string]any)
			if !isMap {
				next = map[string]any{}
				level[part] = next
			}
			level = next
		}
		level[path[len(path)-1]] = value
	}
}
