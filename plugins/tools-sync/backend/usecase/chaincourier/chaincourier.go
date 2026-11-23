// Package chaincourier keeps the chain of the core and the chain of the sync server the same.
package chaincourier

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	publishInterval = time.Second
	adoptEvery      = 5
)

// Deps are what the courier needs.
type Deps struct {
	Identity ports.Identity
	Core     ports.ChainCore
	Chain    ports.Chain
	Endorse  Endorser
}

// Endorser signs the heads the devices of a revoke wrote, so the revoke does not make them unverified.
type Endorser interface {
	Execute(ctx context.Context, revoked [][]byte) error
}

// Publish gives the server the chain the core wants it to hold and tells the core the answer.
type Publish struct{ deps Deps }

// NewPublish returns the use case over the dependencies.
func NewPublish(deps Deps) *Publish { return &Publish{deps: deps} }

// Execute publishes the pending chain of the core: an accepted chain is reported as accepted; where another device appended
// first, the core gets the chain the server holds. It returns whether a chain was pending.
func (u *Publish) Execute(ctx context.Context) (bool, error) {
	pending, err := u.deps.Core.GetChainPublish(ctx)
	if err != nil || !pending.IsPending {
		return false, err
	}
	if err := u.endorse(ctx, pending.Chain); err != nil {
		return false, err
	}
	_, err = u.deps.Chain.AppendChain(ctx, pending.Chain)
	if stale, isStale := errors.AsType[syncprotocol.ChainStaleError](err); isStale {
		return true, u.deps.Core.PutChainPublished(ctx, false, stale.Head)
	}
	if err != nil {
		return false, err
	}
	return true, u.deps.Core.PutChainPublished(ctx, true, nil)
}

// endorse has the devices the pending chain revokes endorsed before the server learns of the revoke: once the chain is on the server
// nothing that those devices wrote counts any more.
func (u *Publish) endorse(ctx context.Context, pending []byte) error {
	chain, err := identity.ParseChain(pending)
	if err != nil {
		return err
	}
	stored, err := u.deps.Chain.GetAccountChain(ctx)
	if err != nil {
		return err
	}
	prefix, err := identity.ParseChain(stored)
	if err != nil || !chain.IsExtensionOf(prefix) {
		return err
	}
	var revoked [][]byte
	for _, device := range chain.RevokedSince(prefix) {
		revoked = append(revoked, device)
	}
	return u.deps.Endorse.Execute(ctx, revoked)
}

// Adopt gives the core the chain of the server when the server holds a longer one than the core.
type Adopt struct{ deps Deps }

// NewAdopt returns the use case over the dependencies.
func NewAdopt(deps Deps) *Adopt { return &Adopt{deps: deps} }

// Execute returns whether the core took a chain.
func (u *Adopt) Execute(ctx context.Context) (bool, error) {
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return false, err
	}
	stored, err := u.deps.Chain.GetAccountChain(ctx)
	if err != nil {
		return false, err
	}
	parsed, err := identity.ParseChain(stored)
	if err != nil || uint32(parsed.Len()) <= own.ChainLength {
		return false, err
	}
	if err := u.deps.Core.AdoptChain(ctx, stored); err != nil {
		return false, err
	}
	return true, nil
}

// Loop runs the two couriers for as long as the plugin runs: a pending chain goes to the server at once, and the chain of the server
// reaches the core every few rounds.
type Loop struct {
	publish *Publish
	adopt   *Adopt
	clock   ports.Clock
}

// NewLoop returns the loop over the dependencies and the clock it waits on.
func NewLoop(deps Deps, clock ports.Clock) *Loop {
	return &Loop{publish: NewPublish(deps), adopt: NewAdopt(deps), clock: clock}
}

// Execute runs until ctx ends and returns its error with the failures of the last round. A failed round is tried again at the next one.
func (u *Loop) Execute(ctx context.Context) error {
	for round := 0; ; round++ {
		_, published := u.publish.Execute(ctx)
		var adopted error
		if round%adoptEvery == 0 {
			_, adopted = u.adopt.Execute(ctx)
		}
		if err := u.clock.Wait(ctx, publishInterval); err != nil {
			return errors.Join(err, published, adopted)
		}
	}
}
