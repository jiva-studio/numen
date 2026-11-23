// Package exitwatch holds the function that is notified when a process that was up exits.
package exitwatch

import "sync"

// Hook holds what is called with the id of a plugin whose process exited and a channel that is closed once
// the process is reaped. The zero value calls nothing.
type Hook struct {
	mu sync.RWMutex
	fn func(pluginID string, exited <-chan struct{})
}

// Set replaces what is called on an exit.
func (h *Hook) Set(fn func(pluginID string, exited <-chan struct{})) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fn = fn
}

// Notify calls what is set, if anything.
func (h *Hook) Notify(pluginID string, exited <-chan struct{}) {
	h.mu.RLock()
	fn := h.fn
	h.mu.RUnlock()
	if fn != nil {
		fn(pluginID, exited)
	}
}
