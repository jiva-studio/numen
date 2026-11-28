package identity

import (
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func receiveAsJoinerLocked(store *linkStore, link *pendingLink, now time.Time, kind byte, body []byte) {
	switch kind {
	case messageRefuse:
		store.endLocked(link, identitydomain.LinkPhaseFailed, decodeRefuse(body))
	case messageGrant:
		receiveGrantLocked(store, link, now, body)
	}
}

func receiveGrantLocked(store *linkStore, link *pendingLink, now time.Time, body []byte) {
	if link.progress.Phase != identitydomain.LinkPhaseWaitingForAccept {
		return
	}
	wrapped, chain, err := decodeGrant(body)
	if err != nil || !readGrant(link, now, wrapped, chain) {
		store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonStaleChain))
		store.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonInvalid)
		return
	}
	link.progress.AccountFingerprint = link.chain.Root().Fingerprint().String()
	store.setPhaseLocked(link, identitydomain.LinkPhaseConfirmAccount)
}

func readGrant(link *pendingLink, now time.Time, wrappedBytes, chainBytes []byte) bool {
	chain, err := identitylib.ParseChain(chainBytes)
	if err != nil || chain.VerifyAt(now) != nil || !chain.IsActiveDevice(link.record.Device.SignerKey()) {
		return false
	}
	wrapped, err := identitylib.ParseWrappedAccountKeys(wrappedBytes)
	if err != nil {
		return false
	}
	account, err := link.record.Device.UnwrapAccountKeys(wrapped)
	if err != nil {
		return false
	}
	if _, err := chain.OpenKeyHistory(account); err != nil {
		return false
	}
	link.record.Wrapped, link.chain = wrapped, chain
	return true
}
