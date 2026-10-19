//go:build !plugins_monolith

package plugin

import "context"

func (h *Host) activatePlugin(ctx context.Context, p DiscoveredPlugin, sup *Supervisor) (Endpoint, error) {
	return h.startPluginProcess(ctx, p, sup)
}
