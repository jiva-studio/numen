package models

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// lockName is the folder that stands for the quota lock, and staleLock is how
// long a lock that is not released stays held.
const (
	lockName  = ".quota.lock"
	staleLock = 30 * time.Second
	lockRetry = 50 * time.Millisecond
)

// touchInterval is how often a live reservation tells the other processes it is
// alive.
var touchInterval = staleReservation / 4

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

// Reservation is the bytes a download has claimed under the quota. It counts
// against every plugin process sharing the folder until Release.
type Reservation struct {
	// Room is the bytes the download may write.
	Room int64

	file string
	stop chan struct{}
	done sync.WaitGroup
	once sync.Once
}

// Reserve claims want bytes of the quota under root for a download, and returns
// an error when they are not there. A negative want claims what is left.
// The bytes the file at part already holds are not counted twice; part may be
// empty.
func Reserve(ctx context.Context, root string, want int64, part string) (*Reservation, error) {
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
	if want < 0 {
		want = left
	}
	if want > left {
		return nil, fmt.Errorf("%s holds %d bytes, and %d more would pass the %d the %s allows", root, used, want, GetQuota(), EnvQuota)
	}
	file, err := os.CreateTemp(root, "reservation-*"+reserveSuffix)
	if err != nil {
		return nil, err
	}
	_, err = file.WriteString(strconv.FormatInt(want, 10) + "\n" + part)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(file.Name())
		return nil, err
	}
	claim := &Reservation{Room: want, file: file.Name(), stop: make(chan struct{})}
	claim.done.Add(1)
	go claim.keepAlive()
	return claim, nil
}

// keepAlive touches the reservation until Release.
func (r *Reservation) keepAlive() {
	defer r.done.Done()
	tick := time.NewTicker(touchInterval)
	defer tick.Stop()
	for {
		select {
		case <-r.stop:
			return
		case now := <-tick.C:
			_ = os.Chtimes(r.file, now, now)
		}
	}
}

// Release gives the claim back.
func (r *Reservation) Release() {
	r.once.Do(func() {
		close(r.stop)
		r.done.Wait()
		os.Remove(r.file)
	})
}
