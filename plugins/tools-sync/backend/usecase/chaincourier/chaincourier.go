// Package chaincourier keeps the chain of the core and the chain of the sync server the same.
package chaincourier

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	publishInterval  = time.Second
	adoptEvery       = 5
	noticeKey        = "device-revoked-others"
	publishFailedKey = "chain-publish-failed"
	// publishFailedText names what failed and not that the plugin cannot sync: the vaults sync on their own.
	publishFailedText = "Could not save the change of the devices on the server"
	maxDetail         = 500
)

// Deps are what the courier needs.
type Deps struct {
	Identity ports.Identity
	Core     ports.ChainCore
	Chain    ports.Chain
	Endorse  Endorser
	Health   ports.Health
	Clock    ports.Clock
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
	if err := u.confirmRevoked(ctx, pending.Chain); err != nil {
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

func (u *Publish) confirmRevoked(ctx context.Context, pending []byte) error {
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
	for _, device := range chain.GetRevokedSince(prefix) {
		revoked = append(revoked, device)
	}
	return u.deps.Endorse.Execute(ctx, revoked)
}

// Adopt gives the core the chain of the server when the server holds a longer one than the core, or one that outranks it at a fork,
// and tells the person when a device removed other devices lately.
type Adopt struct {
	deps     Deps
	isRaised bool
	isClean  bool
}

// NewAdopt returns the use case over the dependencies.
func NewAdopt(deps Deps) *Adopt { return &Adopt{deps: deps, isRaised: false, isClean: false} }

// Execute returns whether the core took a chain.
func (u *Adopt) Execute(ctx context.Context) (bool, error) {
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return false, err
	}
	ownChain, err := identity.ParseChain(own.Chain)
	if err != nil {
		return false, err
	}
	stored, err := u.deps.Chain.GetAccountChain(ctx)
	if err != nil {
		return false, err
	}
	parsed, err := identity.ParseChain(stored)
	if err != nil {
		return false, err
	}
	if parsed.Len() <= ownChain.Len() && !parsed.IsPreferredOver(ownChain) {
		u.notify(ctx, ownChain, own.DeviceSignKey)
		return false, nil
	}
	if err := u.deps.Core.AdoptChain(ctx, stored); err != nil {
		return false, err
	}
	u.notify(ctx, parsed, own.DeviceSignKey)
	return true, nil
}

func (u *Adopt) notify(ctx context.Context, chain identity.Chain, own []byte) {
	now := u.deps.Clock.Now()
	var text string
	for _, revocation := range chain.ListDeviceRevocations() {
		if !bytes.Equal(revocation.By, own) && now.Sub(revocation.At) < identity.CoolingOff {
			text = fmt.Sprintf("A device removed another device from the account on %s. If it was not you, restore with the recovery words to override it", revocation.At.Format("2 Jan 2006"))
		}
	}
	switch {
	case text != "":
		if u.deps.Health.Report(ctx, health.Item{Key: noticeKey, Severity: health.Warning, Text: text}) == nil {
			u.isRaised = true
		}
	case (u.isRaised || !u.isClean) && u.deps.Health.Clear(ctx, noticeKey) == nil:
		u.isRaised, u.isClean = false, true
	}
}

// Loop runs the two couriers for as long as the plugin runs: a pending chain goes to the server at once, and the chain of the server
// reaches the core every few rounds.
type Loop struct {
	publish  *Publish
	adopt    *Adopt
	clock    ports.Clock
	health   ports.Health
	isFailed bool
	isClean  bool
}

// NewLoop returns the loop over the dependencies and the clock it waits on.
func NewLoop(deps Deps, clock ports.Clock) *Loop {
	return &Loop{publish: NewPublish(deps), adopt: NewAdopt(deps), clock: clock, health: deps.Health, isFailed: false, isClean: false}
}

// Execute runs until ctx ends and returns its error with the failures of the last round. A failed round is tried again at the next one;
// a failed publish is logged and shown to the person until a round publishes again.
func (u *Loop) Execute(ctx context.Context) error {
	for round := 0; ; round++ {
		_, published := u.publish.Execute(ctx)
		u.reportPublish(ctx, published)
		var adopted error
		if round%adoptEvery == 0 {
			_, adopted = u.adopt.Execute(ctx)
		}
		if err := u.clock.Wait(ctx, publishInterval); err != nil {
			return errors.Join(err, published, adopted)
		}
	}
}

func (u *Loop) reportPublish(ctx context.Context, published error) {
	if published == nil {
		// The host keeps an item across a restart of the plugin, so the first good round clears it whether or not this run raised it.
		if (u.isFailed || !u.isClean) && u.health.Clear(ctx, publishFailedKey) == nil {
			u.isFailed, u.isClean = false, true
		}
		return
	}
	slog.WarnContext(ctx, "could not publish the chain to the server", "error", published)
	item := health.Item{Key: publishFailedKey, Severity: health.Warning, Text: publishFailedText, Detail: truncate(published.Error(), maxDetail)}
	if u.health.Report(ctx, item) == nil {
		u.isFailed = true
	}
}

func truncate(text string, limit int) string {
	if runes := []rune(text); len(runes) > limit {
		return string(runes[:limit])
	}
	return text
}
