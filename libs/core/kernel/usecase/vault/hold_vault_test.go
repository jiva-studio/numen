package vault_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

type fakeLocks struct {
	mu      sync.Mutex
	held    int
	maximum int
}

func (l *fakeLocks) Lock(_ context.Context, _ domain.Vault) (func(), error) {
	l.mu.Lock()
	l.held++
	l.maximum = max(l.maximum, l.held)
	l.mu.Unlock()
	return func() {
		l.mu.Lock()
		l.held--
		l.mu.Unlock()
	}, nil
}

func (l *fakeLocks) getHeld() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.held
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
	hold := vault.NewHoldVault(locks, &steppedClock{now: time.Unix(1000, 0)})

	id, err := hold.Execute(t.Context(), domain.Vault{})
	if err != nil {
		t.Fatal(err)
	}
	if locks.getHeld() != 1 {
		t.Fatalf("held %d locks after the hold began, want 1", locks.getHeld())
	}
	hold.Release(id)
	hold.Release(id)
	if locks.getHeld() != 0 {
		t.Errorf("held %d locks after the release, want 0", locks.getHeld())
	}
}

func TestAHoldThatEndedIsReleasedWithoutTheHolder(t *testing.T) {
	locks := &fakeLocks{}
	clock := &steppedClock{now: time.Unix(1000, 0)}
	hold := vault.NewHoldVault(locks, clock).WithHold(time.Hour)

	if _, err := hold.Execute(t.Context(), domain.Vault{}); err != nil {
		t.Fatal(err)
	}
	clock.advance(time.Hour - time.Second)
	hold.ReleaseExpired()
	if locks.getHeld() != 1 {
		t.Fatalf("a hold with a second left was released: held %d", locks.getHeld())
	}
	clock.advance(time.Second)
	hold.ReleaseExpired()
	if locks.getHeld() != 0 {
		t.Errorf("held %d locks after the hold ended, want 0", locks.getHeld())
	}
}
