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

// lockFile takes an exclusive inter-process lock on the file at path, waiting
// for the process that holds it until the context ends, and returns what lets it go.
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
