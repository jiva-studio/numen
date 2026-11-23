package identity

import (
	"context"
	"fmt"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const restoreFallback = 30 * time.Second

// StartRestore starts to restore the account of recovery words on this device.
type StartRestore struct{ completion restoreCompletion }

// NewStartRestore returns the use case that starts a restore.
func NewStartRestore(d Deps) *StartRestore { return &StartRestore{completion: d.getCompletion()} }

// Execute parses the words (ErrInvalidWords), refuses when an identity exists (ErrIdentityExists), derives the
// recovery key, makes a device key and keeps both in memory, replacing an earlier pending restore. Nothing is
// written. When no plugin completes the restore within 30 seconds, it completes as a restore with no chain.
func (u *StartRestore) Execute(ctx context.Context, words []string) error {
	parsed, err := identitylib.ParseRecoveryWords(words)
	if err != nil {
		return fmt.Errorf("%w: %w", identitydomain.ErrInvalidWords, err)
	}
	if err := u.completion.records.checkAbsent(ctx); err != nil {
		return err
	}
	recovery, err := parsed.DeriveRecoveryKey()
	if err != nil {
		return err
	}
	clock := u.completion.builder.clock
	device, err := identitylib.NewDeviceKey(u.completion.builder.random)
	if err != nil {
		return err
	}
	restore := &pendingRestore{recovery: recovery, device: device, startedAt: clock.Now(), replaced: make(chan struct{})}
	u.completion.restores.replace(restore)
	go u.completeAfterFallback(context.WithoutCancel(ctx), restore, clock.After(restoreFallback))
	return nil
}

func (u *StartRestore) completeAfterFallback(ctx context.Context, restore *pendingRestore, fired <-chan time.Time) {
	select {
	case <-restore.replaced:
	case <-fired:
		if u.completion.restores.isCurrent(restore) {
			if err := u.completion.complete(ctx, restore, nil); err != nil {
				return // the restore stays pending, so a window or a plugin can complete it
			}
		}
	}
}
