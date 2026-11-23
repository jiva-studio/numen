package identity

import (
	"context"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// SetLinkChannel gives the link the channel the courier made on the relay.
type SetLinkChannel struct{ deps Deps }

// NewSetLinkChannel returns the use case that gives the link its channel.
func NewSetLinkChannel(d Deps) *SetLinkChannel { return &SetLinkChannel{deps: d} }

// Execute makes the key and the secret of the QR code for the channel of 16 bytes. It returns ErrNoLink when no link waits for a channel.
func (u *SetLinkChannel) Execute(_ context.Context, id []byte) error {
	store := u.deps.links
	store.mu.Lock()
	defer store.mu.Unlock()
	store.expireLocked(u.deps.clock.Now())
	link := store.current
	if link == nil || link.isEnded() || !link.relay.wantsChannel {
		return identitydomain.ErrNoLink
	}
	if len(id) != len(identitylib.LinkPayload{}.Channel) {
		return fmt.Errorf("a channel id has %d bytes, got %d", len(identitylib.LinkPayload{}.Channel), len(id))
	}
	displayer, payload, err := identitylib.NewLinkDisplayer(u.deps.random, [16]byte(id), "")
	if err != nil {
		return err
	}
	link.hs.displayer = displayer
	link.relay.channelID = append([]byte(nil), id...)
	link.qrText = payload.Text()
	link.relay.wantsChannel = false
	store.setPhaseLocked(link, identitydomain.LinkPhaseWaitingForPeer)
	return nil
}
