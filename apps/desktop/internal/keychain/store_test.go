package keychain

import (
	"context"
	"time"
)

func newTestStore(dir string) *Store {
	store := NewStore(dir, func(ctx context.Context, d time.Duration) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
			return nil
		}
	})
	store.UsePlaintextFile()
	return store
}
