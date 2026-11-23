package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const restoreFallback = 30 * time.Second

// StartRestore starts to restore the account of recovery words on this device.
type StartRestore struct{ deps Deps }

// NewStartRestore returns the use case that starts a restore.
func NewStartRestore(d Deps) *StartRestore { return &StartRestore{deps: d} }

// Execute parses the words (ErrInvalidWords), refuses when an identity exists (ErrIdentityExists), derives the
// recovery key, makes a device key and keeps both in memory, replacing an earlier pending restore. Nothing is
// written. When no plugin completes the restore within 30 seconds, it completes as a restore with no chain; when the
// key store refuses that write, the restore stays pending and the completion is tried again after 30 seconds.
func (u *StartRestore) Execute(ctx context.Context, words []string) error {
	parsed, err := identitylib.ParseRecoveryWords(words)
	if err != nil {
		return fmt.Errorf("%w: %w", identitydomain.ErrInvalidWords, err)
	}
	if err := checkAbsent(ctx, u.deps); err != nil {
		return err
	}
	recovery, err := parsed.DeriveRecoveryKey()
	if err != nil {
		return err
	}
	device, err := identitylib.NewDeviceKey(u.deps.random)
	if err != nil {
		return err
	}
	restore := &pendingRestore{recovery: recovery, device: device, startedAt: u.deps.clock.Now(), replaced: make(chan struct{})}
	u.deps.pending.replace(restore)
	go u.completeAfterFallback(context.WithoutCancel(ctx), restore, u.deps.clock.After(restoreFallback))
	return nil
}

func (u *StartRestore) completeAfterFallback(ctx context.Context, restore *pendingRestore, fired <-chan time.Time) {
	for {
		select {
		case <-restore.replaced:
			return
		case <-fired:
		}
		if !u.deps.pending.isCurrent(restore) {
			return
		}
		err := completeRestore(ctx, u.deps, restore, nil)
		if !errors.Is(err, identitydomain.ErrKeyStore) {
			return
		}
		fired = u.deps.clock.After(restoreFallback)
	}
}
