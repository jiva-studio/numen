package onnx

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeSession struct {
	id        int
	isRunning atomic.Bool
	destroyed atomic.Int32
}

func newFakePool(t *testing.T, n int) (*sessionPool[*fakeSession], []*fakeSession) {
	t.Helper()
	all := make([]*fakeSession, n)
	for i := range all {
		all[i] = &fakeSession{id: i}
	}
	pool := newSessionPool(all, func(s *fakeSession) {
		if s.isRunning.Load() {
			t.Errorf("session %d destroyed while running", s.id)
		}
		s.destroyed.Add(1)
	})
	return pool, all
}

func TestCloseWaitsForARunInFlightBeforeDestroyingItsSession(t *testing.T) {
	pool, all := newFakePool(t, 2)
	s, err := pool.acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s.isRunning.Store(true)

	closed := make(chan struct{})
	go func() {
		pool.close()
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("Close returned while a run was in flight")
	case <-time.After(50 * time.Millisecond):
	}
	if s.destroyed.Load() != 0 {
		t.Fatal("the session in flight was destroyed")
	}

	s.isRunning.Store(false)
	pool.release(s)
	<-closed
	for _, one := range all {
		if got := one.destroyed.Load(); got != 1 {
			t.Errorf("session %d destroyed %d times", one.id, got)
		}
	}
}

func TestAcquireAfterCloseFailsWithTheClosedError(t *testing.T) {
	pool, _ := newFakePool(t, 1)
	pool.close()
	if _, err := pool.acquire(context.Background()); !errors.Is(err, errClosed) {
		t.Errorf("got %v", err)
	}
	if _, err := pool.acquire(nil); !errors.Is(err, errClosed) {
		t.Errorf("got %v", err)
	}
}

func TestAWaitingAcquireFailsWhenThePoolCloses(t *testing.T) {
	pool, _ := newFakePool(t, 1)
	held, _ := pool.acquire(context.Background())

	result := make(chan error, 1)
	go func() {
		_, err := pool.acquire(context.Background())
		result <- err
	}()
	time.Sleep(20 * time.Millisecond)
	go pool.close()
	if err := <-result; !errors.Is(err, errClosed) {
		t.Errorf("got %v", err)
	}
	pool.release(held)
}

func TestClosingTwiceDestroysEachSessionOnce(t *testing.T) {
	pool, all := newFakePool(t, 3)
	pool.close()
	pool.close()
	for _, one := range all {
		if got := one.destroyed.Load(); got != 1 {
			t.Errorf("session %d destroyed %d times", one.id, got)
		}
	}
}

func TestAnAcquireWithACancelledContextReturnsItsError(t *testing.T) {
	pool, _ := newFakePool(t, 1)
	held, _ := pool.acquire(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := pool.acquire(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
	pool.release(held)
	pool.close()
}

func TestConcurrentAcquireAndCloseNeverTouchADestroyedSession(t *testing.T) {
	before := runtime.NumGoroutine()
	for range 200 {
		pool, _ := newFakePool(t, 2)
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				s, err := pool.acquire(context.Background())
				if err != nil {
					if !errors.Is(err, errClosed) {
						t.Errorf("got %v", err)
					}
					return
				}
				if s.destroyed.Load() != 0 {
					t.Error("lent a destroyed session")
				}
				s.isRunning.Store(true)
				runtime.Gosched()
				s.isRunning.Store(false)
				pool.release(s)
			}()
		}
		wg.Add(2)
		go func() { defer wg.Done(); pool.close() }()
		go func() { defer wg.Done(); pool.close() }()
		wg.Wait()
	}

	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("goroutines went from %d to %d", before, after)
	}
}
