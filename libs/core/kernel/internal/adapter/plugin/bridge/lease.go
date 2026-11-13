package bridge

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Activator starts a plugin on demand and releases the lease the start took. A use lease is for a use of
// the plugin; a question lease is for a question the core asks the plugin for itself, which holds the
// process but does not count as a use of it.
type Activator interface {
	EnsureActive(ctx context.Context, pluginID string, holder plugin.Holder) (transport.Endpoint, error)
	Release(pluginID, token string, holder plugin.Holder)
	EnsureActiveForQuestion(ctx context.Context, pluginID string, holder plugin.Holder) (transport.Endpoint, error)
	ReleaseQuestion(pluginID, token string, holder plugin.Holder)
}

// Lease is the one reference a bridge holds on its plugin, from the moment
// the plugin is activated until the bridge is closed. Token is the token of the
// endpoint the lease was taken on, Holder is who holds it and IsQuestion marks a
// question lease.
type Lease struct {
	Host       Activator
	PluginID   string
	Token      string
	Holder     plugin.Holder
	IsQuestion bool
	once       sync.Once
}

// Close releases the reference, as Release does, and never fails.
func (l *Lease) Close() error {
	l.Release()
	return nil
}

// Release releases the reference. Every call after the first changes nothing.
func (l *Lease) Release() {
	l.once.Do(func() {
		if l.IsQuestion {
			l.Host.ReleaseQuestion(l.PluginID, l.Token, l.Holder)
			return
		}
		l.Host.Release(l.PluginID, l.Token, l.Holder)
	})
}

// Open activates the plugin and returns a client for one of its services
// together with the reference taken on the plugin, held for why. A plugin that cannot be
// activated is a *port.UnavailableError, and the end of the context is returned as
// it is.
func Open[C any](
	ctx context.Context,
	host Activator,
	pluginID string,
	why string,
	newClient func(connect.HTTPClient, string, ...connect.ClientOption) C,
) (C, *Lease, error) {
	return open(ctx, host, pluginID, why, newClient, false)
}

// OpenForQuestion opens the service as Open does, with a question lease held for why: the
// plugin is held for a question the core asks it for itself, which is not a
// use of it.
func OpenForQuestion[C any](
	ctx context.Context,
	host Activator,
	pluginID string,
	why string,
	newClient func(connect.HTTPClient, string, ...connect.ClientOption) C,
) (C, *Lease, error) {
	return open(ctx, host, pluginID, why, newClient, true)
}

func open[C any](
	ctx context.Context,
	host Activator,
	pluginID string,
	why string,
	newClient func(connect.HTTPClient, string, ...connect.ClientOption) C,
	isQuestion bool,
) (C, *Lease, error) {
	var none C
	holder := plugin.GetHolder(ctx, why)
	ensureActive := host.EnsureActive
	if isQuestion {
		ensureActive = host.EnsureActiveForQuestion
	}
	endpoint, err := ensureActive(ctx, pluginID, holder)
	if err != nil {
		return none, nil, ConvertActivationError(pluginID, err)
	}
	interceptor := connect.WithInterceptors(transport.AuthInterceptor(endpoint.Token))
	client := newClient(transport.GetClient(host, endpoint), transport.GetBaseURL(endpoint), interceptor)
	return client, &Lease{Host: host, PluginID: pluginID, Token: endpoint.Token, Holder: holder, IsQuestion: isQuestion}, nil
}

// ConvertActivationError is the failure to activate a plugin: the end of a
// context and a *port.UnavailableError are returned unchanged, and anything else
// is a plugin that failed.
func ConvertActivationError(pluginID string, err error) error {
	var unavailable *plugin.UnavailableError
	if errors.As(err, &unavailable) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return &plugin.UnavailableError{PluginID: pluginID, Reason: plugin.Failed, Err: err}
}

// Service is one service of a plugin, with a client made for the endpoint
// the plugin is at when a call is made. A plugin started again on another port
// is reached there.
type Service[C any] struct {
	Host      Activator
	PluginID  string
	NewClient func(connect.HTTPClient, string, ...connect.ClientOption) C
}

// Open activates the plugin, or finds it running, and returns a client for the
// endpoint it is at now with the reference the call holds until it is closed.
func (s Service[C]) Open(ctx context.Context, why string) (C, *Lease, error) {
	return Open(ctx, s.Host, s.PluginID, why, s.NewClient)
}

// OpenForQuestion holds the plugin for a question the core asks it for itself, which does not mark the
// plugin used or move its idle countdown.
func (s Service[C]) OpenForQuestion(ctx context.Context, why string) (C, *Lease, error) {
	return OpenForQuestion(ctx, s.Host, s.PluginID, why, s.NewClient)
}

// ReadActivationError sorts the failure to start a plugin: the
// end of a context is returned as it is, and any other failure is
// port.ErrUnavailable.
func ReadActivationError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: %w", domain.ErrUnavailable, err)
}
