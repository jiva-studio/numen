package plugin

import "context"

// Refresh sends Configure to each running plugin whose delivery differs from the last one it was
// given, and to each plugin in forced, and waits for every send. An edit of a connection calls it
// after the change is committed. The error names each plugin whose send failed.
// If ctx ends or the call times out the plugin stays due; if the plugin answered with an error a plain refresh does not resend.
func (h *Host) Refresh(ctx context.Context, forced ...string) error {
	return h.configure.refresh(ctx, forced)
}

// WatchPluginConfig registers fn to run each time the plugin has been given its configuration.
// fn runs on its own goroutine, so it may call the plugin while the delivery of Configure is locked.
// The returned function removes the registration.
func (h *Host) WatchPluginConfig(pluginID string, fn func()) (stop func()) {
	return h.configure.watchPluginConfig(pluginID, fn)
}

func (h *Host) listPluginIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.plugins))
	for _, p := range h.getPluginsInOrderLocked() {
		ids = append(ids, p.Manifest.ID)
	}
	return ids
}
