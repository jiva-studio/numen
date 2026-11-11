package hold

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

// Locks holds the writers of a vault for a plugin until the plugin releases them or the hold ends.
type Locks struct {
	locks vaultport.VaultLocks
	clock machineport.Clock
	hold  time.Duration

	mu     sync.Mutex
	leases map[string]hold
	count  int
}

type hold struct {
	release func()
	until   time.Time
}

// NewLocks returns the use case over the locks of the vaults. A hold ends holdFor after it began; LockHold is
// the longest an installation grants.
func NewLocks(locks vaultport.VaultLocks, clock machineport.Clock, holdFor time.Duration) *Locks {
	return &Locks{locks: locks, clock: clock, hold: holdFor, leases: map[string]hold{}}
}

// Execute waits while another holder has the writers of the vault, holds them, and returns the id of
// the hold.
func (u *Locks) Execute(ctx context.Context, v domain.Vault) (string, error) {
	release, err := u.locks.Lock(ctx, v)
	if err != nil {
		return "", err
	}
	u.mu.Lock()
	u.count++
	id := "lock-" + strconv.Itoa(u.count)
	u.leases[id] = hold{release: release, until: u.clock.Now().Add(u.hold)}
	u.mu.Unlock()
	time.AfterFunc(u.hold, u.ReleaseExpired)
	return id, nil
}

// Release gives the writers back. A hold that has ended already is released and changes nothing.
func (u *Locks) Release(id string) {
	u.mu.Lock()
	one, ok := u.leases[id]
	delete(u.leases, id)
	u.mu.Unlock()
	if ok {
		one.release()
	}
}

// ReleaseExpired gives back the writers whose hold has ended.
func (u *Locks) ReleaseExpired() {
	u.mu.Lock()
	var ended []func()
	now := u.clock.Now()
	for id, one := range u.leases {
		if !one.until.After(now) {
			ended = append(ended, one.release)
			delete(u.leases, id)
		}
	}
	u.mu.Unlock()
	for _, release := range ended {
		release()
	}
}
