package pluginport

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// PluginTriggers is the triggers the enabled plugins declare.
type PluginTriggers interface {
	// ListTriggers lists the triggers of every enabled plugin that the host does not hold suspended,
	// in the order of the plugins' ids and then of each manifest.
	ListTriggers() []plugin.TriggerEntry
}
