package identity

import (
	"context"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// linkHost is what the device that has the account does with the messages of the new device: it lists the device in
// the chain, has the server accept the chain and grants the device the account keys.
type linkHost struct {
	records records
	clock   Clock
	random  io.Reader
}

func (h linkHost) receiveLocked(ctx context.Context, store *linkStore, link *pendingLink, kind byte, body []byte) error {
	switch kind {
	case messageJoin:
		return h.receiveJoinLocked(store, link, body)
	case messageCode:
		return h.receiveCodeLocked(ctx, store, link, body)
	case messageRefuse:
		reason, _ := decodeRefuse(body)
		store.endLocked(link, identitydomain.LinkPhaseFailed, reason)
	case messageDone:
		if link.progress.Phase == identitydomain.LinkPhaseConfirmAccount {
			store.endLocked(link, identitydomain.LinkPhaseLinked, "")
		}
	}
	return nil
}

func (h linkHost) receiveJoinLocked(store *linkStore, link *pendingLink, body []byte) error {
	if link.progress.Phase != identitydomain.LinkPhaseWaitingForPeer {
		return nil
	}
	join, err := decodeJoin(body)
	if errors.Is(err, errBadMessage) {
		return nil
	}
	if err != nil {
		return err
	}
	code, err := h.makeCode()
	if err != nil {
		return err
	}
	link.progress.Code, link.progress.TriesLeft = code, maxCodeTries
	link.peer = join.device
	link.progress.PeerName, link.progress.PeerFingerprint = join.name, join.device.Fingerprint().String()
	store.setPhaseLocked(link, identitydomain.LinkPhaseConfirmPeer)
	return nil
}

func (h linkHost) makeCode() (string, error) {
	const limit = 4_294_000_000
	for {
		var raw [4]byte
		if _, err := h.random.Read(raw[:]); err != nil {
			return "", fmt.Errorf("make the confirmation code: %w", err)
		}
		if value := binary.BigEndian.Uint32(raw[:]); value < limit {
			return formatCode(value % 1_000_000), nil
		}
	}
}

func (h linkHost) receiveCodeLocked(ctx context.Context, store *linkStore, link *pendingLink, body []byte) error {
	if !link.progress.IsApproved || link.progress.Phase != identitydomain.LinkPhaseConfirmPeer {
		return nil
	}
	if subtle.ConstantTimeCompare(body, []byte(link.progress.Code)) == 1 {
		return h.publishLocked(ctx, store, link)
	}
	link.progress.TriesLeft--
	if link.progress.TriesLeft <= 0 {
		store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonTooManyWrongCodes, 0))
		store.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonTooManyWrongCodes)
		return nil
	}
	store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonWrongCode, link.progress.TriesLeft))
	return nil
}

func (h linkHost) publishLocked(ctx context.Context, store *linkStore, link *pendingLink) error {
	state, err := h.records.readState(ctx)
	if err != nil {
		return err
	}
	return h.extendLocked(store, link, state)
}

func (h linkHost) extendLocked(store *linkStore, link *pendingLink, state identitylib.State) error {
	extended, isAppended := h.appendPeer(state, link)
	if !isAppended {
		store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonStaleChain, 0))
		store.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonStaleChain)
		return nil
	}
	link.chain = extended
	store.setPhaseLocked(link, identitydomain.LinkPhasePublishing)
	return nil
}

func (h linkHost) appendPeer(state identitylib.State, link *pendingLink) (identitylib.Chain, bool) {
	extended, err := state.Chain.AppendDevice(state.Record.Device, link.peer, link.progress.PeerName, h.clock.Now())
	return extended, err == nil
}

func (h linkHost) answerLocked(ctx context.Context, store *linkStore, link *pendingLink, accepted bool, latest []byte) error {
	state, err := h.records.readState(ctx)
	if err != nil {
		return err
	}
	if accepted {
		return h.grantLocked(ctx, store, link, state)
	}
	state, isAdopted := h.records.adoptLatest(ctx, state, latest)
	if !isAdopted || link.progress.IsRebuilt || state.Status != identitylib.StatusActive {
		store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonStaleChain, 0))
		store.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonStaleChain)
		return nil
	}
	link.progress.IsRebuilt = true
	return h.extendLocked(store, link, state)
}

func (h linkHost) grantLocked(ctx context.Context, store *linkStore, link *pendingLink, state identitylib.State) error {
	next, err := h.records.adoptState(ctx, state, link.chain)
	if err != nil {
		return err
	}
	keys, err := next.OpenAccountKeys()
	if err != nil {
		return err
	}
	wrapped, err := keys.WrapForDevice(link.peer)
	if err != nil {
		return err
	}
	store.sendLocked(link, encodeGrant(wrapped.Bytes(), next.Chain.Bytes()))
	store.setPhaseLocked(link, identitydomain.LinkPhaseConfirmAccount)
	return nil
}
