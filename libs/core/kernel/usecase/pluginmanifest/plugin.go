package pluginmanifest

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// PluginFailures tells the plugins whose start failed.
type PluginFailures interface {
	SetOnFailureCleared(fn func(pluginID string))
	PluginSuspensions
}

// PluginLister returns the metadata of the plugins in the order of their ids.
type PluginLister interface {
	GetPluginInfos() []plugin.Info
	pluginport.PluginEnablement
}

// PluginSuspensions reports whether the host holds a plugin suspended.
type PluginSuspensions interface {
	// IsPluginSuspended reports whether the last start of the plugin failed, which holds it suspended.
	IsPluginSuspended(pluginID string) bool
}
