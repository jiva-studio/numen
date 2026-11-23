package identity

import (
	"errors"
	"io"
)

type LinkPayload struct {
	Version      uint8
	Channel      [16]byte
	DisplayerKey [32]byte
	Secret       [16]byte
	Server       string
}

func (p LinkPayload) Text() string { return "" }

func ParseLinkPayload(text string) (LinkPayload, error) {
	return LinkPayload{}, errors.New("not implemented")
}

// LinkDisplayer is the side of a link that shows the QR code.
type LinkDisplayer struct{}

func NewLinkDisplayer(random io.Reader, channel [16]byte, server string) (*LinkDisplayer, LinkPayload, error) {
	return nil, LinkPayload{}, errors.New("not implemented")
}

func (d *LinkDisplayer) Accept(first []byte) ([]byte, *LinkChannel, error) {
	return nil, nil, errors.New("not implemented")
}

// LinkScanner is the side of a link that reads the QR code.
type LinkScanner struct{}

func NewLinkScanner(random io.Reader, p LinkPayload) (*LinkScanner, []byte, error) {
	return nil, nil, errors.New("not implemented")
}

func (s *LinkScanner) Accept(reply []byte) (*LinkChannel, error) {
	return nil, errors.New("not implemented")
}

// LinkChannel seals and opens the messages of one link in both directions.
type LinkChannel struct{}

func (c *LinkChannel) Seal(plain []byte) []byte { return nil }

func (c *LinkChannel) Open(msg []byte) ([]byte, error) { return nil, errors.New("not implemented") }
