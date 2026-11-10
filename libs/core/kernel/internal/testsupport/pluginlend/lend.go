// Package pluginlend lends the capabilities of a plugin host to a container, for the tests of
// packages outside it.
package pluginlend

import (
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
)

// Lend lends the capabilities of host.
func Lend(host *plugin.Host) *container.PluginHost {
	return container.AdaptPluginHost(host)
}
