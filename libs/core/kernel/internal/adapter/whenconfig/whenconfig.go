// Package whenconfig builds the config variable of when conditions from the settings the plugin host
// resolved, the values and defaults it holds as JSON.
package whenconfig

import (
	"encoding/json"
	"strings"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Settings reads the resolved settings of the application and of the plugins.
type Settings struct {
	App     func() []plugindomain.Setting
	Plugins func() []plugindomain.Info
}

// GetWhenConfig returns the config of the settings App and Plugins give; see New.
func (s Settings) GetWhenConfig() map[string]any {
	var app []plugindomain.Setting
	if s.App != nil {
		app = s.App()
	}
	var plugins []plugindomain.Info
	if s.Plugins != nil {
		plugins = s.Plugins()
	}
	return New(app, plugins)
}

// configAppOwner is the owner of the settings of the application in config.
const configAppOwner = "app"

// New builds the config of when conditions from resolved settings: a setting of the
// application at config.app.<key>, and one of a plugin at config.<plugin id split at its dots>.<key>,
// the key split at its dots too. A setting holds its value, or its default where no value is set; a
// setting with neither is left out.
func New(app []plugindomain.Setting, plugins []plugindomain.Info) map[string]any {
	config := map[string]any{}
	addSettings(config, []string{configAppOwner}, app)
	for _, plugin := range plugins {
		addSettings(config, strings.Split(plugin.ID, "."), plugin.Settings)
	}
	return config
}

func addSettings(config map[string]any, owner []string, settings []plugindomain.Setting) {
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
