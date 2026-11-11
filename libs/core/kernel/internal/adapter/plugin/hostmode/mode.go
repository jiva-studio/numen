// Package hostmode holds the state of the plugin host that every part of it reads: whether it runs in
// safe mode, whether it is shut down, and the vault the paths of the plugins are inside.
package hostmode

import (
	"errors"
	"sync"
)

var (
	// ErrSafeModeActive is what a call fails with while the host runs in safe mode.
	ErrSafeModeActive = errors.New("plugin host is running in safe mode")
	// ErrHostClosed is what a call fails with after the host is shut down.
	ErrHostClosed = errors.New("plugin host is shut down")
)

// Mode holds whether the host runs in safe mode, whether it is shut down, and the vault the paths of
// the plugins are inside.
type Mode struct {
	mu        sync.RWMutex
	isSafe    bool
	isClosed  bool
	vaultPath string
}

// New returns the mode of a host that is running and not in safe mode.
func New() *Mode {
	return &Mode{}
}

// IsSafeMode reports whether the host operates in safe recovery mode.
func (m *Mode) IsSafeMode() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isSafe
}

// IsClosed reports whether the host is shut down.
func (m *Mode) IsClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isClosed
}

// SetVaultPath sets the active vault root path for path traversal checks.
func (m *Mode) SetVaultPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vaultPath = path
}

// GetVaultPath returns the currently configured active vault path.
func (m *Mode) GetVaultPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.vaultPath
}

// SetSafe turns safe mode on or off.
func (m *Mode) SetSafe(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isSafe = enabled
}

// Close marks the host as shut down.
func (m *Mode) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isClosed = true
}
