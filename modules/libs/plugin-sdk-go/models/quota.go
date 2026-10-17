package models

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// lockName is the folder that stands for the quota lock, and staleLock is how
// long a lock that is not released stays held.
const (
	lockName  = ".quota.lock"
	staleLock = 30 * time.Second
	lockRetry = 50 * time.Millisecond
)

// lockQuota takes the lock that serialises the quota decisions of every plugin
// process sharing root.
func lockQuota(ctx context.Context, root string) (func(), error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	lock := filepath.Join(root, lockName)
	for {
		err := os.Mkdir(lock, 0o755)
		if err == nil {
			return func() { os.Remove(lock) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		if info, statErr := os.Stat(lock); statErr == nil && time.Since(info.ModTime()) > staleLock {
			os.Remove(lock)
			continue
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(lockRetry):
		}
	}
}

// reservation is the bytes a download has claimed under the quota.
type reservation struct {
	file string
	room int64
}

// reserveQuota claims want bytes of the quota for the download that writes
// part, and returns an error when they are not there. A want of zero or less
// claims what is left.
func reserveQuota(ctx context.Context, root, part string, want int64) (*reservation, error) {
	unlock, err := lockQuota(ctx, root)
	if err != nil {
		return nil, err
	}
	defer unlock()

	used, err := GetUsage(root)
	if err != nil {
		return nil, fmt.Errorf("counting the bytes under %s: %w", root, err)
	}
	left := max(GetQuota()-used, 0)
	if want <= 0 {
		want = left
	}
	if want > left {
		return nil, fmt.Errorf("%s holds %d bytes, and %d more would pass the %d the %s allows", root, used, want, GetQuota(), EnvQuota)
	}
	file := part + reserveSuffix
	if err := os.WriteFile(file, []byte(strconv.FormatInt(want, 10)), 0o644); err != nil {
		return nil, err
	}
	return &reservation{file: file, room: want}, nil
}

// touch tells the other processes the download is alive.
func (r *reservation) touch() {
	now := time.Now()
	_ = os.Chtimes(r.file, now, now)
}

// release gives the claim back.
func (r *reservation) release() {
	os.Remove(r.file)
}
