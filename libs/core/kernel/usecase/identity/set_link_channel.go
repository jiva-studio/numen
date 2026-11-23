package identity

import (
	"context"
	"fmt"
	"io"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// SetLinkChannel gives the link the channel the courier made on the relay.
type SetLinkChannel struct {
	links  linkHold
	clock  Clock
	random io.Reader
}

// NewSetLinkChannel returns the use case that gives the link its channel.
func NewSetLinkChannel(d Deps) *SetLinkChannel {
	return &SetLinkChannel{links: d.progress.links, clock: d.clock, random: d.random}
}

// Execute makes the key and the secret of the QR code for the channel of 16 bytes. It returns ErrNoLink when no link waits for a channel.
func (u *SetLinkChannel) Execute(_ context.Context, id []byte) error {
	store, release := u.links.hold()
	defer release()
	store.expireLocked(u.clock.Now())
	link := store.current
	if link == nil || link.isEnded() || !link.outbox.WantsChannel {
		return identitydomain.ErrNoLink
	}
	if len(id) != len(identitylib.LinkPayload{}.Channel) {
		return fmt.Errorf("a channel id has %d bytes, got %d", len(identitylib.LinkPayload{}.Channel), len(id))
	}
	displayer, payload, err := identitylib.NewLinkDisplayer(u.random, [16]byte(id), "")
	if err != nil {
		return err
	}
	link.end = &displayerEnd{displayer: displayer}
	link.outbox.ChannelID = append([]byte(nil), id...)
	link.progress.QRText = payload.Text()
	link.outbox.WantsChannel = false
	store.setPhaseLocked(link, identitydomain.LinkPhaseWaitingForPeer)
	return nil
}
