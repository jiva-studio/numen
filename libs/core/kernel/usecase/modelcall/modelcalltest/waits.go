package modelcalltest

import (
	"context"
	"sync"
	"testing"
	"time"
)

type pendingWait struct {
	duration time.Duration
	elapsed  chan struct{}
}

// Waits is a machineport.Wait whose timers end when the test says so.
type Waits struct {
	mu      sync.Mutex
	pending []*pendingWait
	changed chan struct{}
}

func newWaits() *Waits { return &Waits{changed: make(chan struct{}, 256)} }

// Wait holds the caller until Elapse ends its duration or the context ends.
func (w *Waits) Wait(ctx context.Context, d time.Duration) error {
	one := &pendingWait{duration: d, elapsed: make(chan struct{})}
	w.mu.Lock()
	w.pending = append(w.pending, one)
	w.mu.Unlock()
	w.changed <- struct{}{}
	select {
	case <-one.elapsed:
		return nil
	case <-ctx.Done():
		w.mu.Lock()
		for i, other := range w.pending {
			if other == one {
				w.pending = append(w.pending[:i], w.pending[i+1:]...)
				break
			}
		}
		w.mu.Unlock()
		w.changed <- struct{}{}
		return ctx.Err()
	}
}

// List returns the durations of the waits that have not ended, in the order they began.
func (w *Waits) List() []time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	durations := make([]time.Duration, len(w.pending))
	for i, one := range w.pending {
		durations[i] = one.duration
	}
	return durations
}

// AwaitCount blocks until count waits are pending or the test ends.
func (w *Waits) AwaitCount(t *testing.T, count int) {
	t.Helper()
	for len(w.List()) != count {
		select {
		case <-w.changed:
		case <-t.Context().Done():
			t.Fatalf("waits pending = %v, want %d", w.List(), count)
		}
	}
}

// Elapse ends the pending waits of the duration.
func (w *Waits) Elapse(d time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	remaining := w.pending[:0]
	for _, one := range w.pending {
		if one.duration == d {
			close(one.elapsed)
			continue
		}
		remaining = append(remaining, one)
	}
	w.pending = remaining
}
