package synctest

import (
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// WakeLog records the vaults a use case asked the sync loop to pass over.
type WakeLog struct {
	mu     sync.Mutex
	vaults []domain.VaultID
}

// Wake records the vault.
func (w *WakeLog) Wake(vault domain.VaultID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.vaults = append(w.vaults, vault)
}

// Get returns the vaults recorded so far, in order.
func (w *WakeLog) Get() []domain.VaultID {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.vaults)
}

// Count returns how many times the vault was recorded.
func (w *WakeLog) Count(vault domain.VaultID) int {
	return len(slices.DeleteFunc(w.Get(), func(one domain.VaultID) bool { return one != vault }))
}
