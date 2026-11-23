package identity

import (
	"context"
	"fmt"
	"io"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// StartLink starts to link a device: this device is the host that has the account, or the joiner that has none.
type StartLink struct {
	links   linkHold
	records records
	names   DeviceName
	clock   Clock
	random  io.Reader
}

// NewStartLink returns the use case that starts a link.
func NewStartLink(d Deps) *StartLink {
	return &StartLink{links: d.progress.links, records: d.records, names: d.names, clock: d.clock, random: d.random}
}

// Execute starts a link in the mode: show makes a QR code once the courier gave the link a channel, scan reads the text of a QR code.
// A host needs an identity that was not revoked (ErrNoIdentity, ErrDeviceRevoked); a joiner needs none (ErrIdentityExists). A text that is
// not a link is identitylib.ErrLinkPayload. Only one link runs at a time (ErrLinkActive); it lasts ten minutes.
func (u *StartLink) Execute(ctx context.Context, role identitydomain.LinkRole, mode identitydomain.LinkMode, scanned string) (identitydomain.Link, error) {
	store, release := u.links.hold()
	defer release()
	now := u.clock.Now()
	store.expireLocked(now)
	if store.current != nil && !store.current.isEnded() {
		return identitydomain.Link{}, identitydomain.ErrLinkActive
	}
	link, err := u.newLink(ctx, role, mode, now)
	if err != nil {
		return identitydomain.Link{}, err
	}
	if mode == identitydomain.LinkModeScan {
		err = u.readQRText(store, link, scanned)
	} else {
		link.outbox.WantsChannel = true
		link.progress.Phase = identitydomain.LinkPhaseWaitingForChannel
	}
	if err != nil {
		return identitydomain.Link{}, err
	}
	store.current = link
	store.bumpLocked()
	return link.progress.GetLink(), nil
}

func (u *StartLink) newLink(ctx context.Context, role identitydomain.LinkRole, mode identitydomain.LinkMode, now time.Time) (*pendingLink, error) {
	link := &pendingLink{progress: identitydomain.LinkProgress{Role: role, Mode: mode, StartedAt: now}}
	if role == identitydomain.LinkRoleHost {
		return link, u.setHostIdentity(ctx, link)
	}
	if err := u.records.checkAbsent(ctx); err != nil {
		return nil, err
	}
	device, err := identitylib.NewDeviceKey(u.random)
	if err != nil {
		return nil, err
	}
	link.record = identitylib.Record{Device: device}
	link.progress.TriesLeft = maxCodeTries
	link.progress.DeviceName = entryName(u.names.GetInstanceName())
	link.progress.DeviceFingerprint = device.Public().Fingerprint().String()
	return link, nil
}

func (u *StartLink) setHostIdentity(ctx context.Context, link *pendingLink) error {
	state, err := u.records.readState(ctx)
	if err != nil {
		return err
	}
	if state.Status == identitylib.StatusRevoked {
		return identitydomain.ErrDeviceRevoked
	}
	me, err := identityOf(state)
	if err != nil {
		return err
	}
	link.setOwnIdentity(state, me.DeviceName)
	return nil
}

func (u *StartLink) readQRText(store *linkStore, link *pendingLink, scanned string) error {
	payload, err := identitylib.ParseLinkPayload(scanned)
	if err != nil {
		return err
	}
	scanner, first, err := identitylib.NewLinkScanner(u.random, payload)
	if err != nil {
		return fmt.Errorf("start the link: %w", err)
	}
	link.end = &scannerEnd{scanner: scanner}
	link.outbox.ChannelID = append([]byte(nil), payload.Channel[:]...)
	link.outbox.Server = payload.Server
	link.progress.Phase = identitydomain.LinkPhaseWaitingForPeer
	if link.progress.Role == identitydomain.LinkRoleJoiner {
		link.progress.Phase = identitydomain.LinkPhaseWaitingForAccept
	}
	store.queueLocked(link, first)
	return nil
}
