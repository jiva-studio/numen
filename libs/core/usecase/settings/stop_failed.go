package settings

import (
	"context"
	"errors"
)

// ErrNoStopper is a StopFailedPlugin made without a stop.
var ErrNoStopper = errors.New("stop failed plugin: no stopper")

// StopFailedPlugin stops the backend of a plugin whose frontend did not start, and records the plugin as failed.
type StopFailedPlugin struct {
	stop func(ctx context.Context, pluginID, reason string) (*PluginFailure, error)
}

// NewStopFailedPlugin creates the use case over the host's stop. It rejects a nil stop.
func NewStopFailedPlugin(
	stop func(ctx context.Context, pluginID, reason string) (*PluginFailure, error),
) (*StopFailedPlugin, error) {
	if stop == nil {
		return nil, ErrNoStopper
	}
	return &StopFailedPlugin{stop: stop}, nil
}

// Execute stops the backend of the plugin and records it as failed with the reason.
func (u *StopFailedPlugin) Execute(ctx context.Context, pluginID, reason string) (*PluginFailure, error) {
	return u.stop(ctx, pluginID, reason)
}
