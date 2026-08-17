package filesystem

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// locks is one lock per vault root, keyed by the folder on disk. Every writer
// opened on a folder takes the same lock, and so does every opener.
var locks sync.Map

// Hold takes this vault's write lock, waiting for whoever holds it.
//
// The lock is a channel, so a context that ends while it is waited for is
// answered with its error and nothing is held.
func (w Writers) Hold(ctx context.Context, v domain.Vault) (func(), error) {
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
	key := canonical(root)
	if lock, found := locks.Load(key); found {
		return lock.(chan struct{})
	}
	lock, _ := locks.LoadOrStore(key, make(chan struct{}, 1))
	return lock.(chan struct{})
}

// canonical is the one name a folder is locked under. A path that cannot be
// resolved is its own key: what is asked for is what is locked.
func canonical(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return real
}
