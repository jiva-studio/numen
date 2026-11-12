package settings

import (
	"encoding/json"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// WhenConfig reads the resolved settings of the application and of the plugins.
type WhenConfig struct {
	App     func() []plugin.Setting
	Plugins func() []plugin.Info
}

// GetWhenConfig returns the config of the settings App and Plugins give; see BuildWhenConfig.
func (s WhenConfig) GetWhenConfig() map[string]any {
	var app []plugin.Setting
	if s.App != nil {
		app = s.App()
	}
	var plugins []plugin.Info
	if s.Plugins != nil {
		plugins = s.Plugins()
	}
	return BuildWhenConfig(app, plugins)
}

const configAppOwner = "app"

// BuildWhenConfig builds the config of when conditions from resolved settings: a setting of the
// application at config.app.<key>, and one of a plugin at config.<plugin id split at its dots>.<key>,
// the key split at its dots too. A setting holds its value, or its default where no value is set; a
// setting with neither is left out.
func BuildWhenConfig(app []plugin.Setting, plugins []plugin.Info) map[string]any {
	config := map[string]any{}
	addSettings(config, []string{configAppOwner}, app)
	for _, plugin := range plugins {
		addSettings(config, strings.Split(plugin.ID, "."), plugin.Settings)
	}
	return config
}

func addSettings(config map[string]any, owner []string, settings []plugin.Setting) {
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
