package plugin

import (
	"context"
	"net/http"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Runtime lends the bridges to the plugins what they call through, each from the role that has it.
type Runtime struct {
	registry   *Registry
	activation *Activation
	failures   *supervision.Failures
	config     *ConfigPush
}

// EnsureActive starts the plugin if it is not running and holds it.
func (r Runtime) EnsureActive(ctx context.Context, pluginID string) (plugindomain.Endpoint, error) {
	return r.activation.EnsureActive(ctx, pluginID)
}

// Release gives up the hold EnsureActive took.
func (r Runtime) Release(pluginID string) { r.activation.Release(pluginID) }

// GetClient returns the client that calls the process of the endpoint.
func (r Runtime) GetClient(endpoint plugindomain.Endpoint) *http.Client {
	return r.activation.ClientLookup().GetClient(endpoint)
}

// GetPluginInfos lists the plugins the host knows.
func (r Runtime) GetPluginInfos() []plugindomain.Info { return r.registry.GetPluginInfos() }

// GetPlugin returns the enabled plugin with the id.
func (r Runtime) GetPlugin(id string) (manifest.Discovered, bool) { return r.registry.GetPlugin(id) }

// ResolveCapability returns the enabled plugin that declares the capability.
func (r Runtime) ResolveCapability(capability string) (manifest.Discovered, error) {
	return r.registry.ResolveCapability(capability)
}

// ResolveService returns the enabled plugin that serves the service to the windows.
func (r Runtime) ResolveService(service string) (manifest.Discovered, bool) {
	return r.registry.ResolveService(service)
}

// IsPluginSuspended reports whether the last start of the plugin failed.
func (r Runtime) IsPluginSuspended(pluginID string) bool {
	return r.failures.IsPluginSuspended(pluginID)
}

// WatchPluginConfig runs fn each time the plugin has been given its configuration.
func (r Runtime) WatchPluginConfig(pluginID string, fn func()) (stop func()) {
	return r.config.WatchPluginConfig(pluginID, fn)
}
