package provision

import (
	"context"
	"errors"
	"fmt"

	"time"
)

// errLockBusy is the lock being held by another process.
var errLockBusy = errors.New("the lock is held by another process")

const lockPoll = 100 * time.Millisecond

func lockFile(ctx context.Context, path string) (func() error, error) {
	for {
		release, err := acquireFileLock(path)
		if err == nil {
			return release, nil
		}
		if !errors.Is(err, errLockBusy) {
			return nil, fmt.Errorf("lock %s: %w", path, err)
		}
		if err := sleepContext(ctx, lockPoll); err != nil {
			return nil, fmt.Errorf("wait for %s: %w", path, err)
		}
	}
}
