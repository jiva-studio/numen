package pluginport

import "github.com/jiva-studio/numen/libs/plugin-catalog/manifest"

// PluginTabKindEntry is one tab kind an enabled plugin declares, and the plugin that declares it.
type PluginTabKindEntry struct {
	PluginID string
	TabKind  manifest.TabKind
}

// PluginTabKinds is the tab kinds the plugins declare.
type PluginTabKinds interface {
	// ListTabKinds lists the tab kinds of every enabled plugin, in the order of the plugins' ids and
	// then of each manifest.
	ListTabKinds() []PluginTabKindEntry
}
