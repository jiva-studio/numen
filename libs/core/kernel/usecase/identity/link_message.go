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
	messageAccept byte = 2
	messageCode   byte = 3
	messageRefuse byte = 4
	messageGrant  byte = 5
	messageDone   byte = 6
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
	join.name = entryName(string(rest[n:]))
	return join, nil
}

func encodeAccept(isCodeAsked bool) []byte {
	if isCodeAsked {
		return []byte{messageAccept, 1}
	}
	return []byte{messageAccept, 0}
}

func encodeCode(code string) []byte { return append([]byte{messageCode}, code...) }

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

func encodeRefuse(reason identitydomain.LinkReason, triesLeft int) []byte {
	out := []byte{messageRefuse, getReasonCode(reason)}
	if reason == identitydomain.LinkReasonWrongCode {
		out = append(out, byte(triesLeft))
	}
	return out
}

func decodeRefuse(body []byte) (identitydomain.LinkReason, int) {
	if len(body) == 0 {
		return identitydomain.LinkReasonInvalid, 0
	}
	tries := 0
	if len(body) > 1 {
		tries = int(body[1])
	}
	return getReason(body[0]), tries
}

func getReasonCode(reason identitydomain.LinkReason) byte {
	codes := map[identitydomain.LinkReason]byte{
		identitydomain.LinkReasonDeclined:          1,
		identitydomain.LinkReasonDeviceHasIdentity: 2,
		identitydomain.LinkReasonWrongCode:         3,
		identitydomain.LinkReasonTooManyWrongCodes: 4,
		identitydomain.LinkReasonStaleChain:        5,
		identitydomain.LinkReasonCanceled:          6,
	}
	return codes[reason]
}

func getReason(code byte) identitydomain.LinkReason {
	reasons := map[byte]identitydomain.LinkReason{
		1: identitydomain.LinkReasonDeclined,
		2: identitydomain.LinkReasonDeviceHasIdentity,
		3: identitydomain.LinkReasonWrongCode,
		4: identitydomain.LinkReasonTooManyWrongCodes,
		5: identitydomain.LinkReasonStaleChain,
		6: identitydomain.LinkReasonCanceled,
	}
	if reason, ok := reasons[code]; ok {
		return reason
	}
	return identitydomain.LinkReasonInvalid
}
