package shareinvite

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/identity"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
)

var errChannelID = errors.New("the relay made a channel with an id of another size")

// Started is the link a folder is shared by and the moment it stops working.
type Started struct {
	ID        string
	Link      string
	ExpiresAt time.Time
}

// Start makes the link a folder is shared by.
type Start struct{ deps Deps }

// NewStart returns the use case over the dependencies.
func NewStart(deps Deps) *Start { return &Start{deps: deps} }

// Execute makes a channel on the relay and a key for it, and returns the link that holds them. The link works once,
// for as long as the relay keeps the channel and 24 hours at most. The share itself is made when the sharer approves
// the request, so a link nobody uses changes nothing. The folder must be one that may be shared.
func (u *Start) Execute(ctx context.Context, id domain.VaultID, folder string) (Started, error) {
	normalized := domainshare.NormalizeFolder(folder)
	if normalized == "" {
		return Started{}, domainshare.ErrWholeVault
	}
	if _, err := share.NewGetShare(u.deps.Sharing).Execute(ctx, id, normalized); err != nil {
		return Started{}, err
	}
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return Started{}, err
	}
	chain, err := identity.ParseChain(own.Chain)
	if err != nil {
		return Started{}, err
	}
	channel, err := u.deps.Relay.CreateChannel(ctx)
	if err != nil {
		return Started{}, err
	}
	if len(channel.ID) != idSize {
		return Started{}, errChannelID
	}
	expiresAt := channel.ExpiresAt
	if limit := u.deps.Clock.Now().Add(MaxLifetime); expiresAt.After(limit) {
		expiresAt = limit
	}
	displayer, payload, err := identity.NewLinkDisplayer(u.deps.Random, [idSize]byte(channel.ID), u.deps.Server)
	if err != nil {
		return Started{}, fmt.Errorf("make the key of the link: %w", err)
	}
	link, err := domainshare.NewLink(domainshare.LinkInvitation{
		Payload: payload, Folder: path.Base(normalized), Fingerprint: chain.Root().Fingerprint().String(), ExpiresAt: expiresAt,
	})
	if err != nil {
		return Started{}, err
	}
	made := &offer{
		id: channel.ID, vault: id, folder: normalized, expiresAt: expiresAt, displayer: displayer, channel: nil, request: nil, isDone: false,
	}
	u.deps.Hub.mu.Lock()
	u.deps.Hub.offers[hex.EncodeToString(channel.ID)] = made
	u.deps.Hub.mu.Unlock()
	if err := u.deps.Hub.add(ctx, func(run context.Context) { watchOffer(run, u.deps, made) }); err != nil {
		return Started{}, err
	}
	return Started{ID: hex.EncodeToString(channel.ID), Link: link, ExpiresAt: expiresAt}, nil
}
