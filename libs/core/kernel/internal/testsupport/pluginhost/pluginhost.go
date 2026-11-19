// Package pluginhost opens a plugin host the way the container does, and hands out its roles for a test that needs
// more than the capabilities the container lends.
package pluginhost

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
)

// Open returns a plugin host for the target that has discovered the plugins in dirs.
func Open(tb testing.TB, target string, dirs ...string) *Host {
	tb.Helper()
	host := New(container.GetClock(container.NewResources()), target, dirs...)
	err := container.InitializePluginRoles("", container.NewResources(), host.Management, host.Supervision, func(error) {})
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
func Join(host *Host) Roles {
	return Roles{
		Activation: host.Management.Activation, Registry: host.Management.Registry, Settings: host.Management.Settings, ConfigPush: host.Management.Config,
		Failures: host.Supervision.Failures, Events: host.Management.Events, Mode: host.Management.Mode, ContentTypes: host.Management.ContentTypes,
		Discovery: host.Management.Discovery, Services: host.Supervision.Services,
	}
}

// Host holds the two roles exercised by plugin integration tests.
type Host struct {
	Management  *container.PluginManagement
	Supervision *container.PluginSupervision
}

func New(clock machineport.Clock, target string, dirs ...string) *Host {
	management, supervision := container.NewPluginRoles(clock, target, dirs...)
	return &Host{Management: management, Supervision: supervision}
}
