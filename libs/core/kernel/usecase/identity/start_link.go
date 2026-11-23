package identity

import (
	"context"
	"fmt"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// StartLink starts to link a device: this device is the host that has the account, or the joiner that has none.
type StartLink struct{ deps Deps }

// NewStartLink returns the use case that starts a link.
func NewStartLink(d Deps) *StartLink { return &StartLink{deps: d} }

// Execute starts a link in the mode: show makes a QR code once the courier gave the link a channel, scan reads the text of a QR code.
// A host needs an identity that was not revoked (ErrNoIdentity, ErrDeviceRevoked); a joiner needs none (ErrIdentityExists). A text that is
// not a link is identitylib.ErrLinkPayload. Only one link runs at a time (ErrLinkActive); it lasts ten minutes.
func (u *StartLink) Execute(ctx context.Context, role identitydomain.LinkRole, mode identitydomain.LinkMode, scanned string) (identitydomain.Link, error) {
	store := u.deps.links
	store.mu.Lock()
	defer store.mu.Unlock()
	now := u.deps.clock.Now()
	store.expireLocked(now)
	if store.current != nil && !store.current.isEnded() {
		return identitydomain.Link{}, identitydomain.ErrLinkActive
	}
	link, err := u.deps.newLink(ctx, role, mode, now)
	if err != nil {
		return identitydomain.Link{}, err
	}
	if mode == identitydomain.LinkModeScan {
		err = u.deps.readQRText(store, link, scanned)
	} else {
		link.relay.wantsChannel = true
		link.phase = identitydomain.LinkPhaseWaitingForChannel
	}
	if err != nil {
		return identitydomain.Link{}, err
	}
	store.current = link
	store.bumpLocked()
	return link.view(), nil
}

func (d Deps) newLink(ctx context.Context, role identitydomain.LinkRole, mode identitydomain.LinkMode, now time.Time) (*pendingLink, error) {
	link := &pendingLink{role: role, mode: mode, startedAt: now}
	if role == identitydomain.LinkRoleHost {
		return link, d.setHostIdentity(ctx, link)
	}
	if err := checkAbsent(ctx, d); err != nil {
		return nil, err
	}
	device, err := identitylib.NewDeviceKey(d.random)
	if err != nil {
		return nil, err
	}
	link.joiner = joinerSide{device: device, triesLeft: maxCodeTries}
	link.own = ownIdentity{name: entryName(d.names.GetInstanceName()), fingerprint: device.Public().Fingerprint().String()}
	return link, nil
}

func (d Deps) setHostIdentity(ctx context.Context, link *pendingLink) error {
	state, err := d.readState(ctx)
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
	link.own = newOwnIdentity(state, me.DeviceName)
	return nil
}

func (d Deps) readQRText(store *LinkStore, link *pendingLink, scanned string) error {
	payload, err := identitylib.ParseLinkPayload(scanned)
	if err != nil {
		return err
	}
	scanner, first, err := identitylib.NewLinkScanner(d.random, payload)
	if err != nil {
		return fmt.Errorf("start the link: %w", err)
	}
	link.hs.scanner = scanner
	link.relay.channelID = append([]byte(nil), payload.Channel[:]...)
	link.relay.server = payload.Server
	link.phase = identitydomain.LinkPhaseWaitingForPeer
	if link.role == identitydomain.LinkRoleJoiner {
		link.phase = identitydomain.LinkPhaseWaitingForAccept
	}
	store.queueLocked(link, first)
	return nil
}
