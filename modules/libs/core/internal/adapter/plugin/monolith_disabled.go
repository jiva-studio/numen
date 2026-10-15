//go:build !plugins_monolith

package plugin

import "context"

func (h *Host) resolveMonolithPlugin(_ string) (DiscoveredPlugin, bool) {
	return DiscoveredPlugin{}, false
}

func (h *Host) activatePlugin(ctx context.Context, p DiscoveredPlugin, sup *Supervisor) (Endpoint, error) {
	return h.startPluginProcess(ctx, p, sup)
}
