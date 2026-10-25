package platform

import (
	"github.com/jiva-studio/numen/libs/core/container"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// newPluginHost is the plugin host of this binary.
func newPluginHost(target string, dirs ...string) pluginport.PluginManager {
	return container.NewPluginHost(target, dirs...)
}
