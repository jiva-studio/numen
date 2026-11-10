package textbridge

import (
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Host finds the enabled plugins.
type Host interface {
	GetPlugin(id string) (manifest.Discovered, bool)
}

// Plugins is the enabled plugins of the host, which may record text versions.
type Plugins struct {
	host Host
}

// NewPlugins reads the enabled plugins of the host.
func NewPlugins(host Host) Plugins {
	return Plugins{host: host}
}

// IsEnabled reports whether the host has the plugin of the id enabled.
func (p Plugins) IsEnabled(pluginID string) bool {
	_, ok := p.host.GetPlugin(pluginID)
	return ok
}

// NoPlugins is an installation with no plugins.
type NoPlugins struct{}

func (NoPlugins) IsEnabled(string) bool { return false }
