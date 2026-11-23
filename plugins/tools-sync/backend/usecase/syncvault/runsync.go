package syncvault

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	keyUnreachable = "server-unreachable"
	keyFailed      = "sync-failed"
)

// Pass is one pass over one vault.
type Pass interface {
	Execute(ctx context.Context, id domain.VaultID) error
}

// RunDeps are what the loop needs. Interval is read before each wait. Changed names a vault whose files changed.
type RunDeps struct {
	Pass     Pass
	Host     ports.Host
	State    ports.Links
	Health   ports.Health
	Clock    ports.Clock
	Interval func() time.Duration
	Changed  <-chan domain.VaultID
}

// RunSync passes over every connected vault at a fixed interval, and at once over a vault whose files changed.
type RunSync struct{ deps RunDeps }

// NewRunSync returns the use case over the dependencies.
func NewRunSync(deps RunDeps) *RunSync { return &RunSync{deps: deps} }

// Execute runs until ctx ends and returns its error. A server that is away raises a warning and the loop waits the
// same interval again.
func (u *RunSync) Execute(ctx context.Context) error {
	err := u.runLoop(ctx)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return err
}

func (u *RunSync) runLoop(ctx context.Context) error {
	if err := u.passAll(ctx); err != nil {
		return err
	}
	for {
		id, isChange, err := u.wait(ctx)
		if err != nil {
			return err
		}
		if isChange {
			err = u.passOne(ctx, id)
		} else {
			err = u.passAll(ctx)
		}
		if err != nil {
			return err
		}
	}
}

func (u *RunSync) wait(ctx context.Context) (domain.VaultID, bool, error) {
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ended := make(chan error, 1)
	go func() { ended <- u.deps.Clock.Wait(waitCtx, u.deps.Interval()) }()
	select {
	case err := <-ended:
		return "", false, err
	case id := <-u.deps.Changed:
		cancel()
		if err := <-ended; err != nil && !errors.Is(err, context.Canceled) {
			return "", false, err
		}
		return id, true, ctx.Err()
	}
}

func (u *RunSync) passAll(ctx context.Context) error {
	vaults, err := u.deps.Host.GetVaults(ctx)
	if err != nil {
		return u.finish(ctx, err)
	}
	var failure error
	for _, vault := range vaults {
		if err := u.pass(ctx, vault.ID); err != nil {
			failure = err
		}
	}
	return u.finish(ctx, failure)
}

func (u *RunSync) passOne(ctx context.Context, id domain.VaultID) error {
	return u.finish(ctx, u.pass(ctx, id))
}

func (u *RunSync) pass(ctx context.Context, id domain.VaultID) error {
	if _, err := u.deps.State.ReadLink(ctx, id); errors.Is(err, vaultlink.ErrNotConnected) {
		return nil
	}
	return u.deps.Pass.Execute(ctx, id)
}

func (u *RunSync) finish(ctx context.Context, failure error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch {
	case failure == nil:
		return u.clear(ctx)
	case errors.Is(failure, syncprotocol.ErrUnreachable), errors.Is(failure, sessiondomain.ErrUnreachable):
		return u.raise(ctx, keyUnreachable, "Could not reach the sync server")
	case errors.Is(failure, sessiondomain.ErrNoIdentity):
		return nil
	}
	return u.raise(ctx, keyFailed, "Could not sync a vault")
}

func (u *RunSync) raise(ctx context.Context, key, text string) error {
	return u.deps.Health.Report(ctx, health.Item{Key: key, Severity: health.Warning, Text: text, Detail: "", Action: nil})
}

func (u *RunSync) clear(ctx context.Context) error {
	for _, key := range []string{keyUnreachable, keyFailed} {
		if err := u.deps.Health.Clear(ctx, key); err != nil {
			return err
		}
	}
	return nil
}
