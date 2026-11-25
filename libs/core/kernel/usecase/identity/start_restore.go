package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

const (
	restoreFallback = 30 * time.Second
	restoreLifetime = 10 * time.Minute
)

// StartRestore starts to restore the account of recovery words on this device.
type StartRestore struct{ completion restoreCompletion }

// NewStartRestore returns the use case that starts a restore.
func NewStartRestore(d Deps) *StartRestore { return &StartRestore{completion: d.getCompletion()} }

// Execute parses the words (ErrInvalidWords), refuses when an identity exists (ErrIdentityExists), derives the
// recovery key, makes a device key and keeps both in memory, replacing an earlier pending restore. Nothing is
// written. The mark is the chain mark printed on the recovery kit: the restore is refused when the chain of the server does not
// contain it (ErrInvalidChainMark for a text that is no mark). Without a mark the restore needs confirmUnmarked, the explicit confirmation
// of the person, and is refused with ErrInvalidChainMark otherwise. When no plugin completes a confirmed restore within 30 seconds, it
// completes as a restore with no chain, a new identity of the words; when the key store refuses that write, the restore stays pending and the completion is tried
// again after 30 seconds. A pending restore that is not completed within 10 minutes is dropped with its keys.
func (u *StartRestore) Execute(ctx context.Context, words []string, mark string, confirmUnmarked bool) error {
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
	return u.completion.start(ctx, recovery, mark, confirmUnmarked)
}

func readMark(text string) (identitylib.Mark, error) {
	if text == "" {
		return identitylib.Mark{}, nil
	}
	mark, err := identitylib.ParseMark(text)
	if err != nil {
		return identitylib.Mark{}, fmt.Errorf("%w: %w", identitydomain.ErrInvalidChainMark, err)
	}
	return mark, nil
}

func (c restoreCompletion) start(ctx context.Context, recovery identitylib.RecoveryKey, markText string, confirmUnmarked bool) error {
	mark, err := readMark(markText)
	if err != nil {
		return err
	}
	if mark.IsZero() && !confirmUnmarked {
		return fmt.Errorf("%w: the restore names no chain mark and the person did not confirm a restore without one", identitydomain.ErrInvalidChainMark)
	}
	clock := c.builder.clock
	device, err := identitylib.NewDeviceKey(c.builder.random)
	if err != nil {
		return err
	}
	restore := &pendingRestore{recovery: recovery, device: device, mark: mark, startedAt: clock.Now(), replaced: make(chan struct{})}
	c.restores.replace(restore)
	go c.completeAfterFallback(context.WithoutCancel(ctx), restore, clock.After(restoreFallback))
	go c.expireAfterLifetime(restore, clock.After(restoreLifetime))
	return nil
}

func (c restoreCompletion) expireAfterLifetime(restore *pendingRestore, fired <-chan time.Time) {
	select {
	case <-restore.replaced:
	case <-fired:
		c.restores.expire(restore)
	}
}

func (c restoreCompletion) completeAfterFallback(ctx context.Context, restore *pendingRestore, fired <-chan time.Time) {
	for {
		select {
		case <-restore.replaced:
			return
		case <-fired:
		}
		if !c.restores.isCurrent(restore) {
			return
		}
		err := c.complete(ctx, restore, nil)
		if !errors.Is(err, identitydomain.ErrKeyStore) {
			return
		}
		fired = c.builder.clock.After(restoreFallback)
	}
}
