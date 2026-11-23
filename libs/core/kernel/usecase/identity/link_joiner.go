package identity

import (
	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func (d Deps) receiveAsJoinerLocked(link *pendingLink, kind byte, body []byte) error {
	store := d.links
	switch kind {
	case messageAccept:
		if link.phase == identitydomain.LinkPhaseWaitingForAccept && len(body) == 1 && body[0] == 1 {
			store.setPhaseLocked(link, identitydomain.LinkPhaseEnterCode)
		}
	case messageRefuse:
		d.receiveRefuseLocked(link, body)
	case messageGrant:
		d.receiveGrantLocked(link, body)
	}
	return nil
}

func (d Deps) receiveRefuseLocked(link *pendingLink, body []byte) {
	reason, tries := decodeRefuse(body)
	if reason == identitydomain.LinkReasonWrongCode {
		link.joiner.triesLeft = tries
		d.links.setPhaseLocked(link, identitydomain.LinkPhaseEnterCode)
		return
	}
	d.links.endLocked(link, identitydomain.LinkPhaseFailed, reason)
}

func (d Deps) receiveGrantLocked(link *pendingLink, body []byte) {
	if link.phase != identitydomain.LinkPhaseWaitingForAccept {
		return
	}
	wrapped, chain, err := decodeGrant(body)
	if err != nil || !d.readGrantLocked(link, wrapped, chain) {
		d.links.sendLocked(link, encodeRefuse(identitydomain.LinkReasonStaleChain, 0))
		d.links.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonInvalid)
		return
	}
	link.own.account = link.joiner.chain.Root().Fingerprint().String()
	d.links.setPhaseLocked(link, identitydomain.LinkPhaseConfirmAccount)
}

func (d Deps) readGrantLocked(link *pendingLink, wrappedBytes, chainBytes []byte) bool {
	chain, err := identitylib.ParseChain(chainBytes)
	if err != nil || !chain.IsActiveDevice(link.joiner.device.SignerKey()) {
		return false
	}
	wrapped, err := identitylib.ParseWrappedAccountKeys(wrappedBytes)
	if err != nil {
		return false
	}
	if _, err := link.joiner.device.UnwrapAccountKeys(wrapped); err != nil {
		return false
	}
	link.joiner.wrapped, link.joiner.chain = wrapped, chain
	return true
}
