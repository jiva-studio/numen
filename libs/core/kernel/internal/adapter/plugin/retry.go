package plugin

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// pluginActivator starts a plugin and counts the callers that hold it.
type pluginActivator interface {
	EnsureActive(ctx context.Context, pluginID string) (transport.Endpoint, error)
	Release(pluginID string)
}

// listenerRefresher syncs the listeners of a plugin with its settings.
type listenerRefresher interface {
	RefreshListeners(ctx context.Context, pluginID string)
}

// registeredPlugins reports whether the host knows a plugin, and names it.
type registeredPlugins interface {
	isRegistered(pluginID string) bool
	getName(pluginID string) string
}

// Retry starts a plugin once, as any use of it does.
type Retry struct {
	set       registeredPlugins
	failures  *supervision.Failures
	activator pluginActivator
	listeners listenerRefresher
}

func newRetry(set registeredPlugins, failures *supervision.Failures, activator pluginActivator, listeners listenerRefresher) *Retry {
	return &Retry{set: set, failures: failures, activator: activator, listeners: listeners}
}

// RetryPlugin starts a plugin once, as any use of it does, and answers why the start failed where it did.
func (r *Retry) RetryPlugin(ctx context.Context, pluginID string) (*plugindomain.Failure, error) {
	if !r.set.isRegistered(pluginID) {
		return nil, &plugindomain.UnavailableError{PluginID: pluginID, Reason: plugindomain.NotInstalled, Err: errPluginNotFound}
	}
	r.failures.ForgetTargetReport(pluginID)
	defer r.listeners.RefreshListeners(ctx, pluginID)

	_, err := r.activator.EnsureActive(ctx, pluginID)
	if err == nil {
		r.activator.Release(pluginID)
		return nil, nil
	}
	var unavailable *plugindomain.UnavailableError
	if errors.As(err, &unavailable) || ctx.Err() != nil {
		return nil, err
	}
	return r.failures.Get(pluginID, r.set.getName(pluginID)), nil
}
