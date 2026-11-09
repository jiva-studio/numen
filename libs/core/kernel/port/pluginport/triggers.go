package pluginport

import (
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// PluginTriggerEntry is one trigger an enabled plugin declares, and the plugin that declares it.
type PluginTriggerEntry struct {
	PluginID string
	Trigger  manifest.Trigger
}

// PluginTriggers is the triggers the enabled plugins declare.
type PluginTriggers interface {
	// ListTriggers lists the triggers of every enabled plugin that the host does not hold suspended,
	// in the order of the plugins' ids and then of each manifest.
	ListTriggers() []PluginTriggerEntry
}
