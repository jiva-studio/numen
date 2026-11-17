package vaultwriter

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Locks is one write lock per vault root, keyed by the folder on disk. Writers that share a Locks
// take the same lock for a folder.
// Entries are never removed. The key is the folder as the filesystem resolves
// it, so no spelling of a root adds an entry and the number of vaults bounds
// the map; removing one would require counting its waiters.
type Locks struct {
	mu     sync.Mutex
	byRoot map[string]*vaultLock
}

// vaultLock is what every writer of one vault shares. hold is the lock a caller takes through
// VaultWriters.Lock around several steps. A write takes write itself, so a caller that holds hold can
// still write. version is the number of writes the vault has taken: a write raises it under write, and
// a read of it takes no lock.
type vaultLock struct {
	hold    chan struct{}
	write   sync.Mutex
	version atomic.Uint64
}

func newVaultLock() *vaultLock { return &vaultLock{hold: make(chan struct{}, 1)} }

// NewLocks returns a set of locks that no folder holds yet.
func NewLocks() *Locks { return &Locks{byRoot: map[string]*vaultLock{}} }

// ErrNoLocks is returned by writers that were given no Locks to share.
var ErrNoLocks = errors.New("vault writers were given no locks")

// Lock takes this vault's write lock, waiting while another holder has it.
//
// The lock is a channel, so a context that ends while waiting returns its error
// and holds nothing.
func (w VaultWriters) Lock(ctx context.Context, v domain.Vault) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if w.Locks == nil {
		return nil, ErrNoLocks
	}
	lock := w.Locks.getLock(v.Path).hold
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *Locks) getLock(root string) *vaultLock {
	key := getLockKey(root)
	l.mu.Lock()
	defer l.mu.Unlock()
	lock, found := l.byRoot[key]
	if !found {
		lock = newVaultLock()
		l.byRoot[key] = lock
	}
	return lock
}

func getLockKey(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return resolved
}
