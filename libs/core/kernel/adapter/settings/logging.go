package settings

import "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"

const (
	logsPluginID = "md.numen.tools.logs"
	logsLevelKey = "level"
)

// LogLevel returns the severity threshold configured for the Logs plugin, or logs.DefaultLevel when
// the plugin is off or no level is set.
func (f *File) LogLevel() logs.Level {
	if f == nil {
		return logs.DefaultLevel
	}
	pluginValues := f.Plugins[logsPluginID]
	if len(pluginValues) == 0 {
		return logs.DefaultLevel
	}
	if enabled, isBool := pluginValues["enabled"].(bool); !isBool || !enabled {
		return logs.DefaultLevel
	}
	if level, isString := pluginValues[logsLevelKey].(string); isString {
		return logs.ResolveLevel(level)
	}
	return logs.DefaultLevel
}
