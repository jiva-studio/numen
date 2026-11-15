package keychain

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestTwoWritesOfOneStoreRunOneAfterTheOther(t *testing.T) {
	var waitCalls atomic.Int32
	store := NewStore(t.TempDir(), func(ctx context.Context, _ time.Duration) error {
		waitCalls.Add(1)
		return nil
	})
	if cap(store.writer) != 1 {
		t.Fatalf("cap(store.writer) = %d, want 1", cap(store.writer))
	}
	var running, overlaps atomic.Int32
	finished := make(chan error, 2)
	for range 2 {
		go func() {
			finished <- store.changeEntries(t.Context(), func() error {
				if running.Add(1) > 1 {
					overlaps.Add(1)
				}
				time.Sleep(50 * time.Millisecond)
				running.Add(-1)
				return nil
			})
		}()
	}
	for range 2 {
		if err := <-finished; err != nil {
			t.Fatal(err)
		}
	}
	if overlaps.Load() != 0 {
		t.Error("two writes ran at the same time")
	}
	if waitCalls.Load() != 0 {
		t.Errorf("wait was called %d times for contention on the same store, want 0", waitCalls.Load())
	}
}

func TestAWriteWaitingForTheLockReturnsWhenItsContextEnds(t *testing.T) {
	dir := t.TempDir()
	var polled atomic.Int64
	first := newTestStore(dir)
	second := NewStore(dir, func(ctx context.Context, d time.Duration) error {
		polled.Store(int64(d))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
			return nil
		}
	})
	entered, release := make(chan struct{}), make(chan struct{})
	firstResult := make(chan error, 1)
	go func() {
		firstResult <- first.changeEntries(t.Context(), func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	for name, store := range map[string]*Store{"the same store": first, "another store on the folder": second} {
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		err := store.changeEntries(ctx, func() error { return nil })
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("%s: a write behind the lock returned %v, want the deadline", name, err)
		}
	}
	if got := time.Duration(polled.Load()); got != 20*time.Millisecond {
		t.Errorf("lock poll interval = %v, want 20ms", got)
	}

	close(release)
	if err := <-firstResult; err != nil {
		t.Fatal(err)
	}
	if err := second.changeEntries(t.Context(), func() error { return nil }); err != nil {
		t.Errorf("a write after the lock was released: %v", err)
	}
}

func TestLockErrorsReportFailingOperation(t *testing.T) {
	fileAsDir := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(fileAsDir, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := newTestStore(fileAsDir).changeEntries(t.Context(), func() error { return nil })
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) || pathErr.Op != "mkdir" {
		t.Fatalf("changeEntries on file dir = %v, want mkdir PathError", err)
	}

	_, err = lockFile(t.Context(), nil, t.TempDir())
	pathErr = nil
	if !errors.As(err, &pathErr) || pathErr.Op != "open" {
		t.Fatalf("lockFile on directory = %v, want open PathError", err)
	}

	closedFile, err := os.CreateTemp(t.TempDir(), "lock-*")
	if err != nil {
		t.Fatal(err)
	}
	if err := closedFile.Close(); err != nil {
		t.Fatal(err)
	}
	if err := acquireLock(t.Context(), nil, closedFile); err == nil {
		t.Fatal("acquireLock on closed file succeeded")
	}
}
