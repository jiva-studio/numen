package platform

import (
	"github.com/jiva-studio/numen/libs/core/container"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// newPluginHost is the plugin host of this binary.
func newPluginHost(target string, dirs ...string) pluginport.PluginManager {
	return container.NewPluginHost(target, dirs...)
}
