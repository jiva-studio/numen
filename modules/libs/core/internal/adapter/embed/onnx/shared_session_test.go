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

func newSharedPool(t *testing.T, sessions, lanes int) (*sessionPool[*fakeSession], []*fakeSession) {
	t.Helper()
	all := make([]*fakeSession, sessions)
	for i := range all {
		all[i] = &fakeSession{id: i}
	}
	pool := newSessionPool(all, lanes, func(s *fakeSession) {
		if s.isRunning.Load() {
			t.Errorf("session %d destroyed while running", s.id)
		}
		s.destroyed.Add(1)
	})
	return pool, all
}

func TestOneSessionServesAsManyCallersAtOnceAsThePoolHasLanes(t *testing.T) {
	const lanes = 4
	pool, all := newSharedPool(t, 1, lanes)

	var lent []*fakeSession
	for range lanes {
		s, err := pool.acquire(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if s != all[0] {
			t.Fatalf("lent session %d", s.id)
		}
		lent = append(lent, s)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if _, err := pool.acquire(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a caller past the lanes got %v", err)
	}

	pool.release(lent[0])
	s, err := pool.acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	pool.release(s)
	for _, one := range lent[1:] {
		pool.release(one)
	}
	pool.close()
	if got := all[0].destroyed.Load(); got != 1 {
		t.Errorf("the shared session was destroyed %d times", got)
	}
}

func TestLanesAreDealtSessionsInTurn(t *testing.T) {
	pool, all := newSharedPool(t, 2, 5)
	counts := map[*fakeSession]int{}
	var lent []*fakeSession
	for range 5 {
		s, err := pool.acquire(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		counts[s]++
		lent = append(lent, s)
	}
	if counts[all[0]] != 3 || counts[all[1]] != 2 {
		t.Errorf("lanes per session: %d and %d", counts[all[0]], counts[all[1]])
	}
	for _, s := range lent {
		pool.release(s)
	}
	pool.close()
}

func TestAPoolOfNoLanesHasOne(t *testing.T) {
	pool, _ := newSharedPool(t, 1, 0)
	s, err := pool.acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	pool.release(s)
	pool.close()
}

func TestCloseWaitsForEveryLaneOfASharedSession(t *testing.T) {
	pool, all := newSharedPool(t, 1, 3)
	var lent []*fakeSession
	for range 3 {
		s, _ := pool.acquire(t.Context())
		s.isRunning.Store(true)
		lent = append(lent, s)
	}

	closed := make(chan struct{})
	go func() {
		pool.close()
		close(closed)
	}()

	for i, s := range lent {
		select {
		case <-closed:
			t.Fatalf("close returned with %d lanes still lent", len(lent)-i)
		case <-time.After(30 * time.Millisecond):
		}
		if all[0].destroyed.Load() != 0 {
			t.Fatal("the session was destroyed while lent")
		}
		if i == len(lent)-1 {
			s.isRunning.Store(false)
		}
		pool.release(s)
	}
	<-closed
	if got := all[0].destroyed.Load(); got != 1 {
		t.Errorf("destroyed %d times", got)
	}
}

func TestManyCallersNeverExceedTheLanesOfASharedSession(t *testing.T) {
	const lanes, callers, rounds = 3, 12, 40
	before := runtime.NumGoroutine()

	pool, all := newSharedPool(t, 1, lanes)
	var running, peak atomic.Int32
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range rounds {
				s, err := pool.acquire(t.Context())
				if err != nil {
					t.Error(err)
					return
				}
				now := running.Add(1)
				for {
					seen := peak.Load()
					if now <= seen || peak.CompareAndSwap(seen, now) {
						break
					}
				}
				s.isRunning.Store(true)
				runtime.Gosched()
				s.isRunning.Store(false)
				running.Add(-1)
				pool.release(s)
			}
		}()
	}
	wg.Wait()
	pool.close()

	if got := peak.Load(); got > lanes {
		t.Errorf("%d passes ran at once on %d lanes", got, lanes)
	}
	if got := all[0].destroyed.Load(); got != 1 {
		t.Errorf("destroyed %d times", got)
	}
	waitForGoroutines(t, before)
}

func TestACancelledCallerReleasesNoLaneItDidNotTake(t *testing.T) {
	pool, _ := newSharedPool(t, 1, 2)
	first, _ := pool.acquire(t.Context())
	second, _ := pool.acquire(t.Context())

	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() {
		_, err := pool.acquire(ctx)
		result <- err
	}()
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}

	pool.release(first)
	pool.release(second)
	var again []*fakeSession
	for range 2 {
		s, err := pool.acquire(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		again = append(again, s)
	}
	for _, s := range again {
		pool.release(s)
	}
	pool.close()
}

// waitForGoroutines fails the test when goroutines started since before are
// still running a moment later.
func waitForGoroutines(t *testing.T, before int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if now := runtime.NumGoroutine(); now > before {
		t.Errorf("%d goroutines before, %d after", before, now)
	}
}
