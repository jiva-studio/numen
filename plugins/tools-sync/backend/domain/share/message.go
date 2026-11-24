package share

import (
	"encoding/json"
	"errors"
	"fmt"
)

// The kinds of a LinkMessage.
const (
	MessageJoin     = "join"
	MessageApproved = "approved"
	MessageDeclined = "declined"
)

// ErrBadMessage is a message of a link that is not one this version knows.
var ErrBadMessage = errors.New("the message of the link is not one this version knows")

// LinkMessage is what the two sides of a link send each other once the channel is sealed.
type LinkMessage struct {
	Kind       string `json:"k"`
	Chain      []byte `json:"c,omitempty"`
	Name       string `json:"n,omitempty"`
	Invitation string `json:"i,omitempty"`
}

// EncodeLinkMessage returns the bytes sent over the channel.
func EncodeLinkMessage(m LinkMessage) []byte {
	raw, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return raw
}

// DecodeLinkMessage reads the bytes of a message; bytes of another format are ErrBadMessage.
func DecodeLinkMessage(raw []byte) (LinkMessage, error) {
	var m LinkMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return LinkMessage{}, fmt.Errorf("%w: %w", ErrBadMessage, err)
	}
	return m, nil
}
