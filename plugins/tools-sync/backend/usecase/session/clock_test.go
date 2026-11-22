package session_test

import (
	"context"
	"sync"
	"testing"
	"time"
)

// manualClock is the clock of the plugin set by hand: a wait ends when Advance moves the clock past it.
type manualClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []*waiter
	waited  []time.Duration
	arrived chan struct{}
}

type waiter struct {
	end  time.Time
	done chan struct{}
}

func newManualClock(now time.Time) *manualClock {
	return &manualClock{now: now, arrived: make(chan struct{}, 64)}
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) Wait(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	w := &waiter{end: c.now.Add(d), done: make(chan struct{})}
	c.waiters = append(c.waiters, w)
	c.waited = append(c.waited, d)
	c.mu.Unlock()
	c.arrived <- struct{}{}
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Advance moves the clock and ends the waits that fall due.
func (c *manualClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	kept := c.waiters[:0]
	for _, w := range c.waiters {
		if !w.end.After(c.now) {
			close(w.done)
			continue
		}
		kept = append(kept, w)
	}
	c.waiters = kept
}

func (c *manualClock) awaitWait(t *testing.T) time.Duration {
	t.Helper()
	select {
	case <-c.arrived:
	case <-time.After(5 * time.Second):
		t.Fatal("the use case started no wait")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.waited[len(c.waited)-1]
}
