// Package chaincourier keeps the chain of the core and the chain of the sync server the same.
package chaincourier

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	publishInterval  = time.Second
	longestRetryWait = time.Minute
	adoptEvery       = 30
	noticeKey        = "device-revoked-others"
	publishFailedKey = "chain-publish-failed"
	// publishFailedText names what failed and not that the plugin cannot sync: the vaults sync on their own.
	publishFailedText = "Could not save the change of the devices on the server"
	maxDetail         = 500
)

// Deps are what the courier needs.
type Deps struct {
	Identity  ports.Identity
	Core      ports.ChainCore
	Chain     ports.Chain
	Endorse   Endorser
	Health    ports.Health
	Clock     ports.Clock
	IsIdle    func() bool
	Sightings ports.EntrySightings
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
// and tells the person when another device added or removed devices lately.
type Adopt struct {
	deps      Deps
	mu        sync.Mutex
	isRaised  bool
	isClean   bool
	seen      map[string]int64
	isFresh   bool
	isUnsaved bool
}

// NewAdopt returns the use case over the dependencies.
func NewAdopt(deps Deps) *Adopt {
	return &Adopt{deps: deps, mu: sync.Mutex{}, isRaised: false, isClean: false, seen: nil, isFresh: false, isUnsaved: false}
}

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
		return false, u.notify(ctx, ownChain, own.DeviceSignKey)
	}
	if err := u.deps.Core.AdoptChain(ctx, stored); err != nil {
		return false, err
	}
	return true, u.notify(ctx, parsed, own.DeviceSignKey)
}

func (u *Adopt) notify(ctx context.Context, chain identity.Chain, own []byte) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	var text string
	now := u.deps.Clock.Now()
	for _, candidate := range listNotices(chain, own, now, u.recordSightings(ctx, chain, now)) {
		text = candidate
	}
	switch {
	case text != "":
		if u.deps.Health.Report(ctx, health.Item{Key: noticeKey, Severity: health.Warning, Text: text}) == nil {
			u.isRaised = true
		}
	case (u.isRaised || !u.isClean) && u.deps.Health.Clear(ctx, noticeKey) == nil:
		u.isRaised, u.isClean = false, true
	}
	return nil
}

func (u *Adopt) recordSightings(ctx context.Context, chain identity.Chain, now time.Time) map[string]int64 {
	if u.seen == nil {
		stored, err := u.deps.Sightings.ReadEntrySightings(ctx)
		u.seen, u.isFresh = stored, err == nil && stored == nil
		if u.seen == nil {
			u.seen = map[string]int64{}
		}
	}
	isChanged := u.isFresh || u.isUnsaved
	for id, signed := range listEntries(chain) {
		if _, isKnown := u.seen[id]; isKnown {
			continue
		}
		u.seen[id] = now.Unix()
		// The first look of a device with no record takes the entries already in the chain at their signers' time: the person who linked the device saw them.
		if u.isFresh {
			u.seen[id] = min(signed.Unix(), now.Unix())
		}
		isChanged = true
	}
	u.isFresh = false
	if isChanged {
		u.isUnsaved = u.deps.Sightings.WriteEntrySightings(ctx, maps.Clone(u.seen)) != nil
	}
	return u.seen
}

func listEntries(chain identity.Chain) map[string]time.Time {
	found := map[string]time.Time{}
	for _, revocation := range chain.ListDeviceRevocations() {
		found[hex.EncodeToString(revocation.ID[:])] = revocation.At
	}
	for _, addition := range chain.ListDeviceAdditions() {
		found[hex.EncodeToString(addition.ID[:])] = addition.At
	}
	return found
}

func listNotices(chain identity.Chain, own []byte, now time.Time, seen map[string]int64) []string {
	var found []string
	for _, revocation := range chain.ListDeviceRevocations() {
		at, isRecent := getSightingAt(seen, revocation.ID, now)
		if !bytes.Equal(revocation.By, own) && isRecent {
			found = append(found, getRevokedNotice(at))
		}
	}
	for _, addition := range chain.ListDeviceAdditions() {
		at, isRecent := getSightingAt(seen, addition.ID, now)
		if !bytes.Equal(addition.By, own) && !bytes.Equal(addition.Device, own) && isRecent {
			found = append(found, getAddedNotice(at))
		}
	}
	return found
}

func getSightingAt(seen map[string]int64, id [32]byte, now time.Time) (at time.Time, isRecent bool) {
	at = time.Unix(seen[hex.EncodeToString(id[:])], 0).UTC()
	return at, now.Sub(at) < identity.CoolingOff
}

// Loop runs the two couriers for as long as the plugin runs: a pending chain goes to the server at once, and the chain of the server
// reaches the core every few rounds.
type Loop struct {
	publish  *Publish
	adopt    *Adopt
	clock    ports.Clock
	health   ports.Health
	isIdle   func() bool
	isFailed bool
	isClean  bool
}

// NewLoop returns the loop over the dependencies and the clock it waits on.
func NewLoop(deps Deps, clock ports.Clock) *Loop {
	return &Loop{publish: NewPublish(deps), adopt: NewAdopt(deps), clock: clock, health: deps.Health, isIdle: deps.IsIdle, isFailed: false, isClean: false}
}

// Execute runs until ctx ends and returns its error with the failures of the last round. A failed round is tried again at the next one;
// a failed publish is logged and shown to the person until a round publishes again.
func (u *Loop) Execute(ctx context.Context) error {
	wait := publishInterval
	for round := 0; ; round++ {
		published, adopted := u.runRound(ctx, round)
		isFailed := published != nil || adopted != nil
		if !isFailed {
			wait = publishInterval
		}
		if err := u.clock.Wait(ctx, wait); err != nil {
			return errors.Join(err, published, adopted)
		}
		if isFailed {
			wait = min(2*wait, longestRetryWait)
		}
	}
}

func (u *Loop) runRound(ctx context.Context, round int) (published, adopted error) {
	if u.isIdle != nil && u.isIdle() {
		return nil, nil
	}
	_, published = u.publish.Execute(ctx)
	u.reportPublish(ctx, published)
	if round%adoptEvery == 0 {
		_, adopted = u.adopt.Execute(ctx)
	}
	return published, adopted
}

func (u *Loop) reportPublish(ctx context.Context, published error) {
	if published == nil {
		// The host keeps an item across a restart of the plugin, so the first good round clears it whether or not this run raised it.
		if (u.isFailed || !u.isClean) && u.health.Clear(ctx, publishFailedKey) == nil {
			u.isFailed, u.isClean = false, true
		}
		return
	}
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

const noticeDate = "2 Jan 2006"

func getRevokedNotice(at time.Time) string {
	return fmt.Sprintf("A device removed another device from the identity on %s. If it was not you, restore with the recovery words", at.Format(noticeDate))
}

func getAddedNotice(at time.Time) string {
	return fmt.Sprintf("A device added another device to the identity on %s. If it was not you, restore with the recovery words", at.Format(noticeDate))
}
