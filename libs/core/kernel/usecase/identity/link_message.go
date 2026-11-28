package identity

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"unicode/utf8"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// The first byte of a message the two cores send each other through the relay.
const (
	messageJoin   byte = 1
	messageRefuse byte = 2
	messageGrant  byte = 3
	messageDone   byte = 4
)

var errBadMessage = errors.New("not a link message")

type joinRequest struct {
	device identitylib.DevicePublicKey
	name   string
}

func encodeJoin(device identitylib.DevicePublicKey, name string) []byte {
	out := append([]byte{messageJoin}, device.Sign...)
	out = append(out, device.Box[:]...)
	out = binary.AppendUvarint(out, uint64(len(name)))
	return append(out, name...)
}

func decodeJoin(body []byte) (joinRequest, error) {
	if len(body) < ed25519.PublicKeySize+32+1 {
		return joinRequest{}, errBadMessage
	}
	var join joinRequest
	join.device.Sign = ed25519.PublicKey(append([]byte(nil), body[:ed25519.PublicKeySize]...))
	copy(join.device.Box[:], body[ed25519.PublicKeySize:ed25519.PublicKeySize+32])
	rest := body[ed25519.PublicKeySize+32:]
	size, n := binary.Uvarint(rest)
	if n <= 0 || size != uint64(len(rest)-n) || size > maxDeviceName || !utf8.Valid(rest[n:]) {
		return joinRequest{}, errBadMessage
	}
	join.name = normalizeEntryName(string(rest[n:]))
	return join, nil
}

func encodeGrant(wrapped, chain []byte) []byte {
	out := binary.AppendUvarint([]byte{messageGrant}, uint64(len(wrapped)))
	out = append(out, wrapped...)
	return append(out, chain...)
}

func decodeGrant(body []byte) (wrapped, chain []byte, err error) {
	size, n := binary.Uvarint(body)
	if n <= 0 || size > uint64(len(body)-n) {
		return nil, nil, errBadMessage
	}
	return body[n : n+int(size)], body[n+int(size):], nil
}

func encodeRefuse(reason identitydomain.LinkReason) []byte {
	return []byte{messageRefuse, getReasonCode(reason)}
}

func decodeRefuse(body []byte) identitydomain.LinkReason {
	if len(body) == 0 {
		return identitydomain.LinkReasonInvalid
	}
	return getReason(body[0])
}

func getReasonCode(reason identitydomain.LinkReason) byte {
	codes := map[identitydomain.LinkReason]byte{
		identitydomain.LinkReasonDeclined:          1,
		identitydomain.LinkReasonDeviceHasIdentity: 2,
		identitydomain.LinkReasonStaleChain:        3,
		identitydomain.LinkReasonCanceled:          4,
	}
	return codes[reason]
}

func getReason(code byte) identitydomain.LinkReason {
	reasons := map[byte]identitydomain.LinkReason{
		1: identitydomain.LinkReasonDeclined,
		2: identitydomain.LinkReasonDeviceHasIdentity,
		3: identitydomain.LinkReasonStaleChain,
		4: identitydomain.LinkReasonCanceled,
	}
	if reason, ok := reasons[code]; ok {
		return reason
	}
	return identitydomain.LinkReasonInvalid
}
