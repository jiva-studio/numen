package vaultwriter

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// locks is one lock per vault root, keyed by the folder on disk. Every writer
// opened on a folder takes the same lock.
//
// Entries are never removed. It holds one channel per vault root this process has
// written to, and the key is the folder as the filesystem resolves it, so no
// spelling of a root adds a second entry: the number of vaults bounds the map.
// Removing an entry would require counting its waiters, to save a few channels.
var locks = struct {
	sync.Mutex
	activeItem map[string]chan struct{}
}{activeItem: map[string]chan struct{}{}}

// Lock takes this vault's write lock, waiting while another holder has it.
//
// The lock is a channel, so a context that ends while waiting returns its error
// and holds nothing.
func (w VaultWriters) Lock(ctx context.Context, v domain.Vault) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	lock := lockFor(v.Path)
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func lockFor(root string) chan struct{} {
	key := getLockKey(root)
	locks.Lock()
	defer locks.Unlock()
	lock, found := locks.activeItem[key]
	if !found {
		lock = make(chan struct{}, 1)
		locks.activeItem[key] = lock
	}
	return lock
}

// getLockKey returns the name a folder is locked under. A path that cannot be
// resolved is its own key.
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
