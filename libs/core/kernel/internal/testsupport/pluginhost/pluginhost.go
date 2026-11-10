// Package pluginhost opens a plugin host the way the container does, and hands out the host itself
// for a test that needs more than the capabilities the container lends.
package pluginhost

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/libs/core/kernel/container/plugins"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/sharedmimeinfo"
)

// Open returns a plugin host for the target that has discovered the plugins in dirs.
func Open(target string, dirs ...string) *plugin.Host {
	host := plugin.NewHost(clock.System{}, target, dirs...)
	host.SetContentTypeTable(sharedmimeinfo.Open())
	host.SetDependencies(plugins.NewDependenciesIn(""))
	if err := host.Discover(); err != nil {
		fmt.Fprintf(os.Stderr, "numen: discover plugins: %v\n", err)
	}
	return host
}
