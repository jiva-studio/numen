package hold_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/hold"
)

type fakeLocks struct {
	mu      sync.Mutex
	count   int
	maximum int
}

func (l *fakeLocks) Lock(_ context.Context, _ domain.Vault) (func(), error) {
	l.mu.Lock()
	l.count++
	l.maximum = max(l.maximum, l.count)
	l.mu.Unlock()
	return func() {
		l.mu.Lock()
		l.count--
		l.mu.Unlock()
	}, nil
}

func (l *fakeLocks) getCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.count
}

type steppedClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *steppedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *steppedClock) advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

func TestAHoldIsReleasedByItsID(t *testing.T) {
	locks := &fakeLocks{}
	hold := hold.NewLocks(locks, &steppedClock{now: time.Unix(1000, 0)}, hold.LockHold)

	id, err := hold.Execute(t.Context(), domain.Vault{})
	if err != nil {
		t.Fatal(err)
	}
	if locks.getCount() != 1 {
		t.Fatalf("count %d locks after the hold began, want 1", locks.getCount())
	}
	hold.Release(id)
	hold.Release(id)
	if locks.getCount() != 0 {
		t.Errorf("count %d locks after the release, want 0", locks.getCount())
	}
}

func TestAHoldThatEndedIsReleasedWithoutTheHolder(t *testing.T) {
	locks := &fakeLocks{}
	clock := &steppedClock{now: time.Unix(1000, 0)}
	hold := hold.NewLocks(locks, clock, time.Hour)

	if _, err := hold.Execute(t.Context(), domain.Vault{}); err != nil {
		t.Fatal(err)
	}
	clock.advance(time.Hour - time.Second)
	hold.ReleaseExpired()
	if locks.getCount() != 1 {
		t.Fatalf("a hold with a second left was released: count %d", locks.getCount())
	}
	clock.advance(time.Second)
	hold.ReleaseExpired()
	if locks.getCount() != 0 {
		t.Errorf("count %d locks after the hold ended, want 0", locks.getCount())
	}
}
