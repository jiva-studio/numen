package plugin

import (
	"context"
	"errors"
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
// together with the reference held on the plugin. A plugin that cannot be
// activated is a *port.UnavailableError, and the end of the context is kept as
// it is.
func openPlugin[C any](
	ctx context.Context,
	host port.PluginActivator,
	pluginID string,
	newClient func(connect.HTTPClient, string, ...connect.ClientOption) C,
) (C, *pluginLease, error) {
	var none C
	endpoint, err := host.EnsureActive(ctx, pluginID)
	if err != nil {
		return none, nil, readEnsureActiveError(pluginID, err)
	}
	interceptor := connect.WithInterceptors(authInterceptor(endpoint.Token))
	client := newClient(EndpointClient(endpoint), endpoint.BaseURL(), interceptor)
	return client, &pluginLease{host: host, pluginID: pluginID}, nil
}

// readEnsureActiveError is the failure to activate a plugin: the end of a
// context and a *port.UnavailableError are kept as they are, and anything else
// is a plugin that failed.
func readEnsureActiveError(pluginID string, err error) error {
	var unavailable *port.UnavailableError
	if errors.As(err, &unavailable) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return &port.UnavailableError{PluginID: pluginID, Reason: port.PluginFailed, Err: err}
}
