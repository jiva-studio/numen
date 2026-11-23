package client

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type LinkSide uint8

const (
	LinkSideDisplayer LinkSide = 1
	LinkSideScanner   LinkSide = 2
)

type LinkChannel struct {
	ID        []byte
	ExpiresAt time.Time
}

type LinkMessage struct {
	Seq  uint32
	Body []byte
}

type LinkMessages struct {
	Messages []LinkMessage
	IsClosed bool
}

// LinkClient calls the link relay of one server. The relay needs no session.
type LinkClient struct{}

func NewLinkClient(address string, httpClient *http.Client) *LinkClient { return &LinkClient{} }

func (c *LinkClient) CreateChannel(ctx context.Context) (LinkChannel, error) {
	return LinkChannel{}, errors.New("not implemented")
}

func (c *LinkClient) PostMessage(ctx context.Context, channelID []byte, to LinkSide, body []byte) (uint32, error) {
	return 0, errors.New("not implemented")
}

func (c *LinkClient) WaitMessages(ctx context.Context, channelID []byte, reader LinkSide, afterSeq uint32, wait time.Duration) (LinkMessages, error) {
	return LinkMessages{}, errors.New("not implemented")
}

func (c *LinkClient) CloseChannel(ctx context.Context, channelID []byte) error {
	return errors.New("not implemented")
}

// AppendChain presents a longer chain and returns the chain the server stores after the call.
func (c *AccountClient) AppendChain(ctx context.Context, chain []byte) ([]byte, error) {
	return nil, errors.New("not implemented")
}
