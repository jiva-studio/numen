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

func TestAHoldIsReleasedByItsID(t *testing.T) {
	locks := &fakeLocks{}
	hold := hold.NewLocks(locks, hold.LockHold)

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
	hold := hold.NewLocks(locks, 20*time.Millisecond)

	if _, err := hold.Execute(t.Context(), domain.Vault{}); err != nil {
		t.Fatal(err)
	}
	if locks.getCount() != 1 {
		t.Fatalf("count %d locks after the hold began, want 1", locks.getCount())
	}
	deadline := time.Now().Add(5 * time.Second)
	for locks.getCount() != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if locks.getCount() != 0 {
		t.Errorf("count %d locks after the hold ended, want 0", locks.getCount())
	}
}
