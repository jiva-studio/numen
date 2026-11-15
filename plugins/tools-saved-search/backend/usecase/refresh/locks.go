package refresh

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

var errYielded = errors.New("an interactive run of the vault is waiting")

type vaultLocks struct {
	mu    sync.Mutex
	locks map[domain.VaultID]*vaultLock
}

type vaultLock struct {
	isHeld           bool
	interactive      []*lockWaiter
	background       []*lockWaiter
	cancelBackground context.CancelCauseFunc
	batch            chan struct{}
}

type lockWaiter struct {
	ready  chan struct{}
	cancel context.CancelCauseFunc
}

func newVaultLocks() *vaultLocks {
	return &vaultLocks{mu: sync.Mutex{}, locks: map[domain.VaultID]*vaultLock{}}
}

func (l *vaultLocks) list() []domain.VaultID {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Sorted(maps.Keys(l.locks))
}

func (l *vaultLocks) getLocked(vault domain.VaultID) *vaultLock {
	lock, isKnown := l.locks[vault]
	if !isKnown {
		lock = &vaultLock{isHeld: false, interactive: nil, background: nil, cancelBackground: nil, batch: make(chan struct{}, 1)}
		l.locks[vault] = lock
	}
	return lock
}

func (l *vaultLocks) lockBatch(ctx context.Context, vault domain.VaultID) (func(), error) {
	l.mu.Lock()
	batch := l.getLocked(vault).batch
	l.mu.Unlock()
	select {
	case batch <- struct{}{}:
		return func() { <-batch }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *vaultLocks) lockInteractive(ctx context.Context, vault domain.VaultID) (func(), error) {
	l.mu.Lock()
	lock := l.getLocked(vault)
	if !lock.isHeld && len(lock.interactive) == 0 {
		lock.isHeld = true
		l.mu.Unlock()
		return l.newUnlock(lock), nil
	}
	waiter := &lockWaiter{ready: make(chan struct{}), cancel: nil}
	lock.interactive = append(lock.interactive, waiter)
	if lock.cancelBackground != nil {
		lock.cancelBackground(errYielded)
	}
	l.mu.Unlock()
	return l.wait(ctx, lock, waiter)
}

func (l *vaultLocks) lockBackground(ctx context.Context, vault domain.VaultID) (context.Context, func(), error) {
	runCtx, cancel := context.WithCancelCause(ctx)
	l.mu.Lock()
	lock := l.getLocked(vault)
	if !lock.isHeld && len(lock.interactive) == 0 && len(lock.background) == 0 {
		lock.isHeld, lock.cancelBackground = true, cancel
		l.mu.Unlock()
		return runCtx, l.newBackgroundUnlock(lock, cancel), nil
	}
	waiter := &lockWaiter{ready: make(chan struct{}), cancel: cancel}
	lock.background = append(lock.background, waiter)
	l.mu.Unlock()
	if _, err := l.wait(ctx, lock, waiter); err != nil {
		cancel(err)
		return nil, nil, err
	}
	return runCtx, l.newBackgroundUnlock(lock, cancel), nil
}

func (l *vaultLocks) wait(ctx context.Context, lock *vaultLock, waiter *lockWaiter) (func(), error) {
	select {
	case <-waiter.ready:
		return l.newUnlock(lock), nil
	case <-ctx.Done():
	}
	l.mu.Lock()
	isRemoved := removeWaiter(&lock.interactive, waiter) || removeWaiter(&lock.background, waiter)
	l.mu.Unlock()
	if !isRemoved {
		l.unlock(lock)
	}
	return nil, ctx.Err()
}

func (l *vaultLocks) newUnlock(lock *vaultLock) func() {
	var once sync.Once
	return func() { once.Do(func() { l.unlock(lock) }) }
}

func (l *vaultLocks) newBackgroundUnlock(lock *vaultLock, cancel context.CancelCauseFunc) func() {
	unlock := l.newUnlock(lock)
	return func() {
		unlock()
		cancel(context.Canceled)
	}
}

func (l *vaultLocks) unlock(lock *vaultLock) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lock.cancelBackground = nil
	switch {
	case len(lock.interactive) > 0:
		next := lock.interactive[0]
		lock.interactive = lock.interactive[1:]
		close(next.ready)
	case len(lock.background) > 0:
		next := lock.background[0]
		lock.background = lock.background[1:]
		lock.cancelBackground = next.cancel
		close(next.ready)
	default:
		lock.isHeld = false
	}
}

func removeWaiter(waiters *[]*lockWaiter, waiter *lockWaiter) bool {
	i := slices.Index(*waiters, waiter)
	if i < 0 {
		return false
	}
	*waiters = slices.Delete(*waiters, i, i+1)
	return true
}
