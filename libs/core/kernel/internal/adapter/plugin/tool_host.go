package plugin

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// toolActivator starts a plugin for the catalog of tools and holds it.
type toolActivator interface {
	EnsureActive(ctx context.Context, pluginID string, holder plugindomain.Holder) (transport.Endpoint, error)
	Release(pluginID, token string, holder plugindomain.Holder)
	EnsureActiveForQuestion(ctx context.Context, pluginID string, holder plugindomain.Holder) (transport.Endpoint, error)
	ReleaseQuestion(pluginID, token string, holder plugindomain.Holder)
}

// pluginInfoLister lists the plugins the host knows and tells the enabled ones.
type pluginInfoLister interface {
	GetPluginInfos() []plugindomain.Info
	IsEnabled(pluginID string) bool
}

// ToolHost is what the catalog of tools starts plugins and lists them through.
type ToolHost struct {
	activation toolActivator
	registry   pluginInfoLister
}

// NewToolHost returns the host the catalog of tools works over.
func NewToolHost(activation toolActivator, registry pluginInfoLister) ToolHost {
	return ToolHost{activation: activation, registry: registry}
}

// EnsureActive starts the plugin if it is not running and holds it.
func (h ToolHost) EnsureActive(ctx context.Context, pluginID string, holder plugindomain.Holder) (transport.Endpoint, error) {
	return h.activation.EnsureActive(ctx, pluginID, holder)
}

// Release gives up the hold EnsureActive took.
func (h ToolHost) Release(pluginID, token string, holder plugindomain.Holder) {
	h.activation.Release(pluginID, token, holder)
}

// EnsureActiveForQuestion starts the plugin if it is not running and holds it for a question.
func (h ToolHost) EnsureActiveForQuestion(
	ctx context.Context, pluginID string, holder plugindomain.Holder,
) (transport.Endpoint, error) {
	return h.activation.EnsureActiveForQuestion(ctx, pluginID, holder)
}

// ReleaseQuestion gives up the hold EnsureActiveForQuestion took.
func (h ToolHost) ReleaseQuestion(pluginID, token string, holder plugindomain.Holder) {
	h.activation.ReleaseQuestion(pluginID, token, holder)
}

// GetPluginInfos lists the plugins the host knows.
func (h ToolHost) GetPluginInfos() []plugindomain.Info { return h.registry.GetPluginInfos() }

// IsEnabled is false for a disabled plugin and for every plugin in safe mode.
func (h ToolHost) IsEnabled(pluginID string) bool { return h.registry.IsEnabled(pluginID) }
