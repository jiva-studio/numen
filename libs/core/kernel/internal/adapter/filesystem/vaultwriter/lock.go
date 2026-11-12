package vaultwriter

import (
	"context"
	"errors"
	"path/filepath"
	"sync"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Locks is one write lock per vault root, keyed by the folder on disk. Writers that share a Locks
// take the same lock for a folder.
// Entries are never removed. The key is the folder as the filesystem resolves
// it, so no spelling of a root adds an entry and the number of vaults bounds
// the map; removing one would require counting its waiters.
type Locks struct {
	mu     sync.Mutex
	byRoot map[string]chan struct{}
}

// NewLocks returns a set of locks that no folder holds yet.
func NewLocks() *Locks { return &Locks{byRoot: map[string]chan struct{}{}} }

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
	lock := w.Locks.getLock(v.Path)
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *Locks) getLock(root string) chan struct{} {
	key := getLockKey(root)
	l.mu.Lock()
	defer l.mu.Unlock()
	lock, found := l.byRoot[key]
	if !found {
		lock = make(chan struct{}, 1)
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
