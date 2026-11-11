package plugin

import "sync"

// Mode holds whether the host runs in safe mode, whether it is shut down, and the vault the paths of
// the plugins are inside.
type Mode struct {
	mu        sync.RWMutex
	isSafe    bool
	isClosed  bool
	vaultPath string
}

// NewMode creates the mode of a host that is neither in safe mode nor shut down, with no vault.
func NewMode() *Mode {
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

func (m *Mode) setSafe(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isSafe = enabled
}

func (m *Mode) close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isClosed = true
}
