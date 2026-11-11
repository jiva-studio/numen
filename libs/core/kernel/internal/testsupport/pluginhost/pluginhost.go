// Package pluginhost opens a plugin host the way the container does, and hands out its roles for a test that needs
// more than the capabilities the container lends.
package pluginhost

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Host is the roles of a plugin host, the ones the container holds, so that a test lends it to the container with
// container.AdaptPluginHost.
type Host = container.PluginRoles

// Open returns a plugin host for the target that has discovered the plugins in dirs.
func Open(tb testing.TB, target string, dirs ...string) *Host {
	tb.Helper()
	host, err := container.OpenPluginRoles(target, dirs...)
	if err != nil {
		tb.Logf("discover plugins: %v", err)
	}
	return host
}

// New creates the roles of a plugin host for the target application and the folders its plugins are in, as the
// container makes them. The roles read the time from clock.
func New(clock machineport.Clock, target string, dirs ...string) *Host {
	return container.NewPluginRoles(clock, target, dirs...)
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
func Join(host *Host) Roles {
	return Roles{
		Activation: host.Activation, Registry: host.Registry, Settings: host.Settings, ConfigPush: host.Config,
		Failures: host.Failures, Events: host.Events, Mode: host.Mode, ContentTypes: host.ContentTypes,
		Discovery: host.Discovery, Services: host.Services,
	}
}
