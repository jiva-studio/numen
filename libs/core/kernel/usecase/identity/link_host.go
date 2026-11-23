package identity

import (
	"context"
	"crypto/subtle"
	"encoding/binary"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func (d Deps) receiveAsHostLocked(ctx context.Context, link *pendingLink, kind byte, body []byte) error {
	store := d.links
	switch kind {
	case messageJoin:
		return d.receiveJoinLocked(link, body)
	case messageCode:
		return d.receiveCodeLocked(ctx, link, body)
	case messageRefuse:
		reason, _ := decodeRefuse(body)
		store.endLocked(link, identitydomain.LinkPhaseFailed, reason)
	case messageDone:
		if link.phase == identitydomain.LinkPhaseConfirmAccount {
			store.endLocked(link, identitydomain.LinkPhaseLinked, "")
		}
	}
	return nil
}

func (d Deps) receiveJoinLocked(link *pendingLink, body []byte) error {
	if link.phase != identitydomain.LinkPhaseWaitingForPeer {
		return nil
	}
	join, err := decodeJoin(body)
	if err != nil {
		return nil
	}
	if link.mode == identitydomain.LinkModeScan {
		code, err := d.makeCode()
		if err != nil {
			return err
		}
		link.host.code, link.host.triesLeft = code, maxCodeTries
	}
	link.host.join = join
	d.links.setPhaseLocked(link, identitydomain.LinkPhaseConfirmPeer)
	return nil
}

func (d Deps) makeCode() (string, error) {
	const limit = 4_294_000_000
	for {
		var raw [4]byte
		if _, err := d.random.Read(raw[:]); err != nil {
			return "", fmt.Errorf("make the confirmation code: %w", err)
		}
		if value := binary.BigEndian.Uint32(raw[:]); value < limit {
			return formatCode(value % 1_000_000), nil
		}
	}
}

func (d Deps) receiveCodeLocked(ctx context.Context, link *pendingLink, body []byte) error {
	store := d.links
	if !link.host.isApproved || link.phase != identitydomain.LinkPhaseConfirmPeer {
		return nil
	}
	if subtle.ConstantTimeCompare(body, []byte(link.host.code)) == 1 {
		return d.publishLinkLocked(ctx, link)
	}
	link.host.triesLeft--
	if link.host.triesLeft <= 0 {
		store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonTooManyWrongCodes, 0))
		store.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonTooManyWrongCodes)
		return nil
	}
	store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonWrongCode, link.host.triesLeft))
	return nil
}

func (d Deps) publishLinkLocked(ctx context.Context, link *pendingLink) error {
	state, err := d.readState(ctx)
	if err != nil {
		return err
	}
	return d.extendLocked(link, state)
}

func (d Deps) extendLocked(link *pendingLink, state identitylib.State) error {
	extended, err := state.Chain.AppendDevice(state.Record.Device, link.host.join.device, link.host.join.name, d.clock.Now())
	if err != nil {
		d.links.sendLocked(link, encodeRefuse(identitydomain.LinkReasonStaleChain, 0))
		d.links.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonStaleChain)
		return nil
	}
	link.host.extended = extended
	d.links.setPhaseLocked(link, identitydomain.LinkPhasePublishing)
	return nil
}
