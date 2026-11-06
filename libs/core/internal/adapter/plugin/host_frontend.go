package plugin

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// StopFailedPlugin stops the process of a plugin whose frontend did not start in a window, and records the
// plugin as failed with the reason, so the host does not start it again until it is retried. A plugin with no
// process running is only recorded, and one without a backend is not recorded at all.
func (h *Host) StopFailedPlugin(ctx context.Context, pluginID, reason string) (*pluginport.PluginFailure, error) {
	h.mu.RLock()
	_, recognized := h.plugins[pluginID]
	h.mu.RUnlock()
	if !recognized {
		return nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginNotInstalled, Err: errPluginNotFound}
	}
	if err := h.stopPlugin(ctx, pluginID); err != nil {
		return nil, err
	}
	failure := h.recordActivationFailure(pluginID, errors.New(reason))
	h.deliverPluginFailures()
	return failure, nil
}
