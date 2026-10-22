package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

const pluginFailureTaskPrefix = "plugin-failed:"

// GetPluginRetrier starts, as the window asks, a plugin the host suspends once more.
func (c Config) GetPluginRetrier() func(ctx context.Context, pluginID string) (*pluginport.PluginFailure, error) {
	return func(ctx context.Context, pluginID string) (*pluginport.PluginFailure, error) {
		if c.PluginHost == nil {
			return nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginNotInstalled}
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
	c.PluginHost.SetOnPluginFailed(func(failure pluginport.PluginFailure) {
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
