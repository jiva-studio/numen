package shareinvite

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	kindJoin     = "join"
	kindApproved = "approved"
	kindDeclined = "declined"
)

var errBadMessage = errors.New("the message of the link is not one this version knows")

// message is what the two sides send each other once the channel is sealed.
type message struct {
	Kind       string `json:"k"`
	Chain      []byte `json:"c,omitempty"`
	Name       string `json:"n,omitempty"`
	Invitation string `json:"i,omitempty"`
}

func encodeMessage(m message) []byte {
	raw, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return raw
}

func decodeMessage(raw []byte) (message, error) {
	var m message
	if err := json.Unmarshal(raw, &m); err != nil {
		return message{}, fmt.Errorf("%w: %w", errBadMessage, err)
	}
	return m, nil
}
