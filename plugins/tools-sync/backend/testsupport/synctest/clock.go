// Package synctest is the world the tests of Numen Sync run in: a sync server in memory behind a switch,
// computers that each hold an identity, vaults and a health host, and the clock they share. It wires the
// real adapters and use cases of the plugin over the real fake server and the test hosts of the SDK.
package synctest

import (
	"context"
	"sync"
	"testing"
	"time"
)

type Clock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []*waiter
	arrived chan time.Duration
}

type waiter struct {
	end  time.Time
	done chan struct{}
}

func NewClock(now time.Time) *Clock {
	return &Clock{mu: sync.Mutex{}, now: now, waiters: nil, arrived: make(chan time.Duration, 64)}
}

func GetStart() time.Time {
	return time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
}

func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *Clock) Wait(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	w := &waiter{end: c.now.Add(d), done: make(chan struct{})}
	c.waiters = append(c.waiters, w)
	c.mu.Unlock()
	select {
	case c.arrived <- d:
	default:
	}
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Clock) Advance(d time.Duration) {
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

func (c *Clock) Set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
}

func (c *Clock) AwaitWait(tb testing.TB) time.Duration {
	tb.Helper()
	return c.AwaitWaitWithin(tb, 5*time.Second)
}

func (c *Clock) AwaitWaitWithin(tb testing.TB, limit time.Duration) time.Duration {
	tb.Helper()
	select {
	case d := <-c.arrived:
		return d
	case <-time.After(limit):
		tb.Fatal("the use case started no wait")
		return 0
	}
}

func WaitUntil(tb testing.TB, what string, condition func() bool) {
	tb.Helper()
	timeout := time.After(5 * time.Second)
	for !condition() {
		select {
		case <-timeout:
			tb.Fatalf("timed out waiting for %s", what)
		case <-time.After(5 * time.Millisecond):
		}
	}
}
