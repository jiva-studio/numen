package keychain

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestTwoWritesOfOneStoreRunOneAfterTheOther(t *testing.T) {
	store := NewStore(t.TempDir())
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
}

func TestAWriteWaitingForTheLockReturnsWhenItsContextEnds(t *testing.T) {
	dir := t.TempDir()
	first, second := NewStore(dir), NewStore(dir)
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

	close(release)
	if err := <-firstResult; err != nil {
		t.Fatal(err)
	}
	if err := second.changeEntries(t.Context(), func() error { return nil }); err != nil {
		t.Errorf("a write after the lock was released: %v", err)
	}
}
