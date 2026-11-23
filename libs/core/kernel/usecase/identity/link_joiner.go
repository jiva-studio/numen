package identity

import (
	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func receiveAsJoinerLocked(store *linkStore, link *pendingLink, kind byte, body []byte) {
	switch kind {
	case messageAccept:
		if link.progress.Phase == identitydomain.LinkPhaseWaitingForAccept && len(body) == 1 && body[0] == 1 {
			store.setPhaseLocked(link, identitydomain.LinkPhaseEnterCode)
		}
	case messageRefuse:
		receiveRefuseLocked(store, link, body)
	case messageGrant:
		receiveGrantLocked(store, link, body)
	}
}

func receiveRefuseLocked(store *linkStore, link *pendingLink, body []byte) {
	reason, tries := decodeRefuse(body)
	if reason == identitydomain.LinkReasonWrongCode {
		link.progress.TriesLeft = tries
		store.setPhaseLocked(link, identitydomain.LinkPhaseEnterCode)
		return
	}
	store.endLocked(link, identitydomain.LinkPhaseFailed, reason)
}

func receiveGrantLocked(store *linkStore, link *pendingLink, body []byte) {
	if link.progress.Phase != identitydomain.LinkPhaseWaitingForAccept {
		return
	}
	wrapped, chain, err := decodeGrant(body)
	if err != nil || !readGrant(link, wrapped, chain) {
		store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonStaleChain, 0))
		store.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonInvalid)
		return
	}
	link.progress.AccountFingerprint = link.chain.Root().Fingerprint().String()
	store.setPhaseLocked(link, identitydomain.LinkPhaseConfirmAccount)
}

func readGrant(link *pendingLink, wrappedBytes, chainBytes []byte) bool {
	chain, err := identitylib.ParseChain(chainBytes)
	if err != nil || !chain.IsActiveDevice(link.record.Device.SignerKey()) {
		return false
	}
	wrapped, err := identitylib.ParseWrappedAccountKeys(wrappedBytes)
	if err != nil {
		return false
	}
	if _, err := link.record.Device.UnwrapAccountKeys(wrapped); err != nil {
		return false
	}
	link.record.Wrapped, link.chain = wrapped, chain
	return true
}
