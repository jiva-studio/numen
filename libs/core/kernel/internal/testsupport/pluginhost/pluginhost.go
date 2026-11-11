// Package pluginhost opens a plugin host the way the container does, and hands out its roles for a test that needs
// more than the capabilities the container lends.
package pluginhost

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
)

// Open returns a plugin host for the target that has discovered the plugins in dirs.
func Open(tb testing.TB, target string, dirs ...string) *container.PluginRoles {
	tb.Helper()
	host, err := container.OpenPluginRoles(func(error) {}, target, dirs...)
	if err != nil {
		tb.Logf("discover plugins: %v", err)
	}
	return host
}

// Roles presents the roles of a host as one value, for a test that gives a bridge the whole host.
type Roles struct {
	*plugin.Activation
	*plugin.Registry
	*plugin.Settings
	*plugin.ConfigPush
	*supervision.Failures
	*plugin.Events
	*plugin.Mode
	*plugin.ContentTypes
	*plugin.Discovery
	*supervision.Services
}

// Join presents the roles of host as one value.
func Join(host *container.PluginRoles) Roles {
	return Roles{
		Activation: host.Activation, Registry: host.Registry, Settings: host.Settings, ConfigPush: host.Config,
		Failures: host.Failures, Events: host.Events, Mode: host.Mode, ContentTypes: host.ContentTypes,
		Discovery: host.Discovery, Services: host.Services,
	}
}
