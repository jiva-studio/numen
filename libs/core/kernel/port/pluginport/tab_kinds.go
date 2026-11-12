package pluginport

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

type PluginTabKinds interface {
	// ListTabKinds lists the tab kinds of every enabled plugin, in the order of the plugins' ids and
	// then of each manifest.
	ListTabKinds() []plugin.TabKindEntry
}
