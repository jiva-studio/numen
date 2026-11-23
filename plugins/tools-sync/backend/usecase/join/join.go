// Package join adds this device to the account of the recovery words a person typed, by reading the chain of
// that account from the server for the host.
package join

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const watchInterval = 5 * time.Second

// Identity is the restore the host keeps until the plugin completes it.
type Identity interface {
	GetPendingRestore(ctx context.Context) (session.PendingRestore, error)
	SignAsRecovery(ctx context.Context, message []byte) ([]byte, error)
	CompleteRestore(ctx context.Context, chain []byte) error
}

// Server is the sync server, which gives the chain of an account to the holder of its recovery key.
type Server interface {
	GetChainReadChallenge(ctx context.Context, rootSignKey []byte) (session.Challenge, error)
	GetAccountChainByRoot(ctx context.Context, challengeID, rootSignKey, signature []byte) ([]byte, bool, error)
}

// Deps are what Join needs.
type Deps struct {
	Identity Identity
	Server   Server
}

// Join completes a pending restore.
type Join struct{ deps Deps }

// NewJoin returns the use case over the dependencies.
func NewJoin(deps Deps) *Join { return &Join{deps: deps} }

// Execute completes the pending restore with the chain of the account, or with none when the server has no such
// account, and reports whether there was one to complete.
func (u *Join) Execute(ctx context.Context) (bool, error) {
	pending, err := u.deps.Identity.GetPendingRestore(ctx)
	if err != nil || !pending.IsPending {
		return false, err
	}
	challenge, err := u.deps.Server.GetChainReadChallenge(ctx, pending.RootSignKey)
	if err != nil {
		return false, err
	}
	signature, err := u.deps.Identity.SignAsRecovery(ctx, challenge.Message)
	if err != nil {
		return false, ignoreNoRestore(err)
	}
	chain, found, err := u.deps.Server.GetAccountChainByRoot(ctx, challenge.ID, pending.RootSignKey, signature)
	if err != nil {
		return false, err
	}
	if !found {
		chain = nil
	}
	if err := u.deps.Identity.CompleteRestore(ctx, chain); err != nil {
		return false, ignoreNoRestore(err)
	}
	return true, nil
}

func ignoreNoRestore(err error) error {
	if errors.Is(err, session.ErrNoPendingRestore) {
		return nil
	}
	return err
}

// Watch completes a restore as soon as one is pending.
type Watch struct {
	join  *Join
	clock ports.Clock
}

// NewWatch returns the use case over a join and a clock.
func NewWatch(join *Join, clock ports.Clock) *Watch { return &Watch{join: join, clock: clock} }

// Execute looks for a pending restore every five seconds until ctx ends, and returns its error. A restore that
// cannot be completed now is looked at again.
func (u *Watch) Execute(ctx context.Context) error {
	for {
		if _, err := u.join.Execute(ctx); err != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		if err := u.clock.Wait(ctx, watchInterval); err != nil {
			return err
		}
	}
}
