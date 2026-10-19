package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

const pluginFailureTaskPrefix = "plugin-failed:"

// GetPluginRetrier starts, as the window asks, a plugin the host suspends once more.
func (c Config) GetPluginRetrier() func(ctx context.Context, pluginID string) (*port.PluginFailure, error) {
	return func(ctx context.Context, pluginID string) (*port.PluginFailure, error) {
		if c.PluginHost == nil {
			return nil, &port.UnavailableError{PluginID: pluginID, Reason: port.PluginNotInstalled}
		}
		return c.PluginHost.RetryPlugin(ctx, pluginID)
	}
}

// ReportPluginFailures keeps one task in the list for each plugin the host suspends, with the reason as its
// error, and takes it out when the host forgets the failure.
func (c Config) ReportPluginFailures(tasks *run.Tasks) {
	if c.PluginHost == nil {
		return
	}
	c.PluginHost.SetOnPluginFailed(func(failure port.PluginFailure) {
		tasks.Set(run.Task{
			ID:    pluginFailureTaskPrefix + failure.PluginID,
			Title: "Plugin stopped",
			About: failure.Name,
			Error: failure.Reason,
		})
	})
	c.PluginHost.SetOnFailureCleared(func(pluginID string) {
		tasks.Remove(pluginFailureTaskPrefix + pluginID)
	})
}
