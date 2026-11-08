package keychain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// lockSuffix is added to the name of the credentials file to name the file its writers lock, which
// lies beside the credentials folder. The credentials file cannot be locked, since every write
// renames a new file over it.
const lockSuffix = ".lock"

func (s *Store) changeEntries(ctx context.Context, work func() error) (err error) {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("lock the credentials file: %w", err)
	}
	release, err := s.locks.lock(ctx, filepath.Join(s.dir, credentialsFile+lockSuffix))
	if err != nil {
		return fmt.Errorf("lock the credentials file: %w", err)
	}
	defer func() { err = errors.Join(err, release()) }()
	s.mu.Lock()
	defer s.mu.Unlock()
	return work()
}

// lockWait is the one wait for the lock on a file that every caller waiting for it shares.
type lockWait struct {
	ended   chan struct{}
	file    *os.File
	err     error
	waiters int
	isEnded bool
	isTaken bool
}

// lockRegistry holds the wait in progress for each lock file of a store. The system call that waits
// cannot be stopped, so a caller that stops waiting leaves it to the others: a lock file has at most
// one blocked goroutine and one open descriptor, however many callers cancel.
type lockRegistry struct {
	mu    sync.Mutex
	waits map[string]*lockWait
}

func (r *lockRegistry) lock(ctx context.Context, path string) (func() error, error) {
	for {
		wait := r.join(path)
		select {
		case <-wait.ended:
			file, isTaken, err := r.take(wait)
			if err != nil {
				return nil, err
			}
			if !isTaken {
				continue
			}
			if err := ctx.Err(); err != nil {
				return nil, errors.Join(err, file.Close())
			}
			return file.Close, nil
		case <-ctx.Done():
			r.leave(wait)
			return nil, ctx.Err()
		}
	}
}

func (r *lockRegistry) join(path string) *lockWait {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.waits == nil {
		r.waits = make(map[string]*lockWait)
	}
	wait := r.waits[path]
	if wait == nil {
		wait = &lockWait{ended: make(chan struct{})}
		r.waits[path] = wait
		go r.run(path, wait)
	}
	wait.waiters++
	return wait
}

func (r *lockRegistry) run(path string, wait *lockWait) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err == nil {
		if err = lockOpenFile(file); err != nil {
			err = errors.Join(err, file.Close())
			file = nil
		}
	}
	r.mu.Lock()
	wait.file, wait.err, wait.isEnded = file, err, true
	if r.waits[path] == wait {
		delete(r.waits, path)
	}
	if wait.waiters == 0 {
		wait.releaseUntaken()
	}
	r.mu.Unlock()
	close(wait.ended)
}

func (r *lockRegistry) take(wait *lockWait) (file *os.File, isTaken bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	wait.waiters--
	if wait.err != nil {
		return nil, false, wait.err
	}
	if wait.isTaken || wait.file == nil {
		return nil, false, nil
	}
	wait.isTaken = true
	return wait.file, true, nil
}

func (r *lockRegistry) leave(wait *lockWait) {
	r.mu.Lock()
	defer r.mu.Unlock()
	wait.waiters--
	if wait.isEnded && wait.waiters == 0 && !wait.isTaken {
		wait.releaseUntaken()
	}
}

func (w *lockWait) releaseUntaken() {
	if w.file == nil {
		return
	}
	w.err = errors.Join(w.err, w.file.Close())
	w.file = nil
}
