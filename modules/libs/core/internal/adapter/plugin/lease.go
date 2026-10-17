package plugin

import (
	"context"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// pluginLease is the one reference a bridge holds on its plugin, from the moment
// the plugin is activated until the bridge is closed.
type pluginLease struct {
	host     port.PluginActivator
	pluginID string
	once     sync.Once
}

// Close releases the reference. Every call after the first changes nothing.
func (l *pluginLease) Close() error {
	l.once.Do(func() {
		l.host.Release(l.pluginID)
	})
	return nil
}

// openPlugin activates the plugin and returns a client for one of its services
// together with the reference held on the plugin.
func openPlugin[C any](
	ctx context.Context,
	host port.PluginActivator,
	pluginID string,
	newClient func(connect.HTTPClient, string, ...connect.ClientOption) C,
) (C, *pluginLease, error) {
	var none C
	endpoint, err := host.EnsureActive(ctx, pluginID)
	if err != nil {
		return none, nil, fmt.Errorf("ensure active %s: %w", pluginID, err)
	}
	interceptor := connect.WithInterceptors(authInterceptor(endpoint.Token))
	client := newClient(EndpointClient(endpoint), endpoint.BaseURL(), interceptor)
	return client, &pluginLease{host: host, pluginID: pluginID}, nil
}
