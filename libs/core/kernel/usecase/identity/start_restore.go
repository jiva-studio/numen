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
type StartRestore struct{ deps Deps }

// NewStartRestore returns the use case that starts a restore.
func NewStartRestore(d Deps) *StartRestore { return &StartRestore{deps: d} }

// Execute parses the words (ErrInvalidWords), refuses when an identity exists (ErrIdentityExists), derives the
// recovery key, makes a device key and keeps both in memory, replacing an earlier pending restore. Nothing is
// written. When no plugin completes the restore within 30 seconds, it completes as a restore with no chain.
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
	return startPending(ctx, u.deps, recovery)
}

func startPending(ctx context.Context, deps Deps, recovery identitylib.RecoveryKey) error {
	device, err := identitylib.NewDeviceKey(deps.random)
	if err != nil {
		return err
	}
	restore := &pendingRestore{recovery: recovery, device: device, startedAt: deps.clock.Now(), replaced: make(chan struct{})}
	deps.pending.replace(restore)
	go completeAfterFallback(context.WithoutCancel(ctx), deps, restore, deps.clock.After(restoreFallback))
	return nil
}

func completeAfterFallback(ctx context.Context, deps Deps, restore *pendingRestore, fired <-chan time.Time) {
	select {
	case <-restore.replaced:
	case <-fired:
		if deps.pending.isCurrent(restore) {
			if err := completeRestore(ctx, deps, restore, nil); err != nil {
				return // the restore stays pending, so a window or a plugin can complete it
			}
		}
	}
}
