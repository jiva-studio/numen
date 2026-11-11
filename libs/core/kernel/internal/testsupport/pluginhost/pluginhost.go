// Package pluginhost opens a plugin host the way the container does, and hands out the host itself
// for a test that needs more than the capabilities the container lends.
package pluginhost

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/onnxruntime"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/provision"
)

// Open returns a plugin host for the target that has discovered the plugins in dirs.
func Open(tb testing.TB, target string, dirs ...string) *plugin.Host {
	tb.Helper()
	host := plugin.NewHost(clock.System{}, target, dirs...)
	host.ContentTypes.SetContentTypeTable(contenttype.Open())
	host.Services.SetDependencies(dependencies())
	if err := host.Discovery.Discover(); err != nil {
		tb.Logf("discover plugins: %v", err)
	}
	return host
}

func dependencies() *provision.Registry {
	runtimeConfig := onnxruntime.FromEnvironment(clock.System{})
	runtimeConfig.Dir = onnxruntime.GetModelsDir(runtimeConfig.Dir)
	return provision.NewRegistry(
		onnxruntime.NewONNXProvisioner(runtimeConfig),
		provision.NewFileProvisioner(provision.Config{Dir: runtimeConfig.Dir, Offline: onnxruntime.IsOffline(), Clock: clock.System{}}),
	)
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
func Join(host *plugin.Host) Roles {
	return Roles{
		Activation: host.Activation, Registry: host.Registry, Settings: host.Settings, ConfigPush: host.Config,
		Failures: host.Failures, Events: host.Events, Mode: host.Mode, ContentTypes: host.ContentTypes,
		Discovery: host.Discovery, Services: host.Services,
	}
}
