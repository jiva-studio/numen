package keychain

import (
	"context"
	"time"
)

// NewSystemStore returns a store that waits on the timers of this machine where it polls for the lock of its file.
func NewSystemStore(dir string) *Store {
	return NewStore(dir, waitOnTimer)
}

func waitOnTimer(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
