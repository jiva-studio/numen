package plugin

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
)

// Mode holds whether the host runs in safe mode and the vault the plugins' paths are inside.
type Mode struct {
	t         *table
	errs      *errorSink
	teardown  *Teardown
	listeners *Listeners
}

func newMode(t *table, errs *errorSink, teardown *Teardown, listeners *Listeners) *Mode {
	return &Mode{t: t, errs: errs, teardown: teardown, listeners: listeners}
}

// SetSafeMode enables or disables safe recovery mode. Entering it stops every running plugin.
func (m *Mode) SetSafeMode(enabled bool) {
	m.t.mu.Lock()
	m.t.safeMode = enabled
	var running []string
	if enabled {
		running = m.getRunningPluginIDsLocked()
	}
	m.t.mu.Unlock()

	for _, id := range running {
		if err := m.teardown.StopPlugin(context.Background(), id); err != nil {
			m.errs.report(err)
		}
	}
	m.listeners.syncAll(context.Background())
}

func (m *Mode) getRunningPluginIDsLocked() []string {
	ids := make([]string, 0, len(m.t.supervisors))
	for id, sup := range m.t.supervisors {
		if sup.GetState() != lifecycle.StateDormant {
			ids = append(ids, id)
		}
	}
	return ids
}

// IsSafeMode reports whether the host operates in safe recovery mode.
func (m *Mode) IsSafeMode() bool {
	m.t.mu.RLock()
	defer m.t.mu.RUnlock()
	return m.t.safeMode
}

// SetVaultPath sets the active vault root path for path traversal checks.
func (m *Mode) SetVaultPath(path string) {
	m.t.mu.Lock()
	defer m.t.mu.Unlock()
	m.t.vaultPath = path
}

// GetVaultPath returns the currently configured active vault path.
func (m *Mode) GetVaultPath() string {
	m.t.mu.RLock()
	defer m.t.mu.RUnlock()
	return m.t.vaultPath
}
