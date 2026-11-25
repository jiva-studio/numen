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
)

var errChannelID = errors.New("the relay made a channel with an id of another size")

// SharedLink is the link a folder is shared by and the moment it stops working.
type SharedLink struct {
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
// the request, so a link that no person uses changes nothing. The folder must be one that may be shared.
func (u *Start) Execute(ctx context.Context, id domain.VaultID, folder string) (SharedLink, error) {
	normalized := domainshare.NormalizeFolder(folder)
	if err := u.deps.Sharing.CheckShareable(ctx, id, normalized); err != nil {
		return SharedLink{}, err
	}
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return SharedLink{}, err
	}
	name, err := u.getFolderName(ctx, id, normalized)
	if err != nil {
		return SharedLink{}, err
	}
	chain, err := identity.ParseChain(own.Chain)
	if err != nil {
		return SharedLink{}, err
	}
	channel, err := u.deps.Relay.CreateChannel(ctx)
	if err != nil {
		return SharedLink{}, err
	}
	if len(channel.ID) != idSize {
		return SharedLink{}, errChannelID
	}
	expiresAt := channel.ExpiresAt
	if limit := u.deps.Clock.Now().Add(MaxLifetime); expiresAt.After(limit) {
		expiresAt = limit
	}
	displayer, payload, err := identity.NewLinkDisplayer(u.deps.Random, [idSize]byte(channel.ID), u.deps.Server)
	if err != nil {
		return SharedLink{}, fmt.Errorf("make the key of the link: %w", err)
	}
	link, err := domainshare.NewLink(domainshare.LinkInvitation{
		Payload: payload, Folder: name, Fingerprint: chain.Root().Fingerprint().String(), ExpiresAt: expiresAt,
	})
	if err != nil {
		return SharedLink{}, err
	}
	made := &offer{
		id: channel.ID, vault: id, folder: normalized, name: name, expiresAt: expiresAt, displayer: displayer, channel: nil, request: nil, isDone: false,
	}
	u.deps.Hub.mu.Lock()
	u.deps.Hub.offers[hex.EncodeToString(channel.ID)] = made
	u.deps.Hub.mu.Unlock()
	if err := u.deps.Hub.add(ctx, func(run context.Context) { watchOffer(run, u.deps, made) }); err != nil {
		return SharedLink{}, err
	}
	return SharedLink{ID: hex.EncodeToString(channel.ID), Link: link, ExpiresAt: expiresAt}, nil
}

func (u *Start) getFolderName(ctx context.Context, id domain.VaultID, folder string) (string, error) {
	if folder == "" {
		return u.deps.Sharing.GetVaultName(ctx, id)
	}
	return path.Base(folder), nil
}
