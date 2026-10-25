package bridge

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// Lease is the one reference a bridge holds on its plugin, from the moment
// the plugin is activated until the bridge is closed.
type Lease struct {
	Host     pluginport.PluginActivator
	PluginID string
	once     sync.Once
}

// Close releases the reference. Every call after the first changes nothing.
func (l *Lease) Close() error {
	l.once.Do(func() {
		l.Host.Release(l.PluginID)
	})
	return nil
}

// Open activates the plugin and returns a client for one of its services
// together with the reference taken on the plugin. A plugin that cannot be
// activated is a *port.UnavailableError, and the end of the context is returned as
// it is.
func Open[C any](
	ctx context.Context,
	host pluginport.PluginActivator,
	pluginID string,
	newClient func(connect.HTTPClient, string, ...connect.ClientOption) C,
) (C, *Lease, error) {
	var none C
	endpoint, err := host.EnsureActive(ctx, pluginID)
	if err != nil {
		return none, nil, ConvertActivationError(pluginID, err)
	}
	interceptor := connect.WithInterceptors(transport.AuthInterceptor(endpoint.Token))
	client := newClient(transport.EndpointClient(endpoint), endpoint.BaseURL(), interceptor)
	return client, &Lease{Host: host, PluginID: pluginID}, nil
}

// ConvertActivationError is the failure to activate a plugin: the end of a
// context and a *port.UnavailableError are returned unchanged, and anything else
// is a plugin that failed.
func ConvertActivationError(pluginID string, err error) error {
	var unavailable *pluginport.UnavailableError
	if errors.As(err, &unavailable) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginFailed, Err: err}
}

// Service is one service of a plugin, with a client made for the endpoint
// the plugin is at when a call is made. A plugin started again on another port
// is reached there.
type Service[C any] struct {
	Host      pluginport.PluginActivator
	PluginID  string
	NewClient func(connect.HTTPClient, string, ...connect.ClientOption) C
}

// Open activates the plugin, or finds it running, and returns a client for the
// endpoint it is at now with the reference the call holds until it is closed.
func (s Service[C]) Open(ctx context.Context) (C, *Lease, error) {
	return Open(ctx, s.Host, s.PluginID, s.NewClient)
}

// ReadActivationError sorts the failure to start the proofreading plugin: the
// end of a context is returned as it is, and any other failure is
// port.ErrUnavailable.
func ReadActivationError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: %w", completionport.ErrUnavailable, err)
}
