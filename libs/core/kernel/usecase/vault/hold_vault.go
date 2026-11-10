package vault

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// LockHold is the longest a plugin holds the writers of a vault. A plugin that goes without releasing
// them must not stop the vault being written.
const LockHold = 2 * time.Minute

// HoldVault holds the writers of a vault for a plugin until the plugin releases them or the hold ends.
type HoldVault struct {
	locks vaultport.VaultLocks
	clock machineport.Clock
	hold  time.Duration

	mu    sync.Mutex
	held  map[string]hold
	count int
}

type hold struct {
	release func()
	until   time.Time
}

// NewHoldVault returns the use case over the locks of the vaults. A hold ends LockHold after it began.
func NewHoldVault(locks vaultport.VaultLocks, clock machineport.Clock) *HoldVault {
	return &HoldVault{locks: locks, clock: clock, hold: LockHold, held: map[string]hold{}}
}

// WithHold returns the use case with holds that end after d.
func (u *HoldVault) WithHold(d time.Duration) *HoldVault {
	u.hold = d
	return u
}

// Execute waits while another holder has the writers of the vault, holds them, and returns the id of
// the hold.
func (u *HoldVault) Execute(ctx context.Context, v domain.Vault) (string, error) {
	release, err := u.locks.Lock(ctx, v)
	if err != nil {
		return "", err
	}
	u.mu.Lock()
	u.count++
	id := "lock-" + strconv.Itoa(u.count)
	u.held[id] = hold{release: release, until: u.clock.Now().Add(u.hold)}
	u.mu.Unlock()
	time.AfterFunc(u.hold, u.ReleaseExpired)
	return id, nil
}

// Release gives the writers back. A hold that has ended already is released and changes nothing.
func (u *HoldVault) Release(id string) {
	u.mu.Lock()
	one, ok := u.held[id]
	delete(u.held, id)
	u.mu.Unlock()
	if ok {
		one.release()
	}
}

// ReleaseExpired gives back the writers whose hold has ended.
func (u *HoldVault) ReleaseExpired() {
	u.mu.Lock()
	var ended []func()
	now := u.clock.Now()
	for id, one := range u.held {
		if !one.until.After(now) {
			ended = append(ended, one.release)
			delete(u.held, id)
		}
	}
	u.mu.Unlock()
	for _, release := range ended {
		release()
	}
}
