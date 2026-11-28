package synctest

import (
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

type WakeLog struct {
	mu     sync.Mutex
	vaults []domain.VaultID
}

func (w *WakeLog) Wake(vault domain.VaultID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.vaults = append(w.vaults, vault)
}

func (w *WakeLog) Get() []domain.VaultID {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.vaults)
}

func (w *WakeLog) Count(vault domain.VaultID) int {
	return len(slices.DeleteFunc(w.Get(), func(one domain.VaultID) bool { return one != vault }))
}
