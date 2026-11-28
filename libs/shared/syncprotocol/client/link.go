package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

// LinkSide is one of the two devices of a link.
type LinkSide uint8

const (
	// LinkSideDisplayer is the device that shows the QR code.
	LinkSideDisplayer LinkSide = 1
	// LinkSideScanner is the device that reads it.
	LinkSideScanner LinkSide = 2
)

// LinkChannel is a channel of the relay.
type LinkChannel struct {
	ID        []byte
	ExpiresAt time.Time
}

// LinkMessage is a sealed message of the relay, numbered in the order it was posted.
type LinkMessage struct {
	Seq  uint32
	Body []byte
}

// LinkMessages is the answer of a wait: the messages, and whether the channel ended.
type LinkMessages struct {
	Messages []LinkMessage
	IsClosed bool
}

// LinkClient calls the link relay of one server. The relay needs no session.
type LinkClient struct {
	rpc syncv1connect.LinkServiceClient
}

// NewLinkClient returns the client of the relay of the server at address.
func NewLinkClient(address string, httpClient *http.Client) *LinkClient {
	options := connect.WithInterceptors(syncprotocol.NewVersionInterceptor())
	return &LinkClient{rpc: syncv1connect.NewLinkServiceClient(httpClient, address, options)}
}

// CreateChannel makes a channel that lives ten minutes.
func (c *LinkClient) CreateChannel(ctx context.Context) (LinkChannel, error) {
	response, err := c.rpc.CreateChannel(ctx, connect.NewRequest(&syncv1.CreateChannelRequest{}))
	if err != nil {
		return LinkChannel{}, readLinkError(err)
	}
	return LinkChannel{ID: response.Msg.GetChannelId(), ExpiresAt: time.Unix(response.Msg.GetExpiresAt(), 0)}, nil
}

// SendMessage stores a message for the side and returns its number. A channel that is not there is ErrChannelGone.
func (c *LinkClient) SendMessage(ctx context.Context, channelID []byte, to LinkSide, body []byte) (uint32, error) {
	response, err := c.rpc.SendMessage(ctx, connect.NewRequest(&syncv1.SendMessageRequest{
		ChannelId: channelID, To: writeLinkSide(to), Body: body,
	}))
	if err != nil {
		return 0, readLinkError(err)
	}
	return response.Msg.GetSeq(), nil
}

// WaitMessages returns the messages for the reader after afterSeq, waiting up to wait, at most 20 seconds, for one.
// A second wait on the same side is ErrLinkBusy.
func (c *LinkClient) WaitMessages(ctx context.Context, channelID []byte, reader LinkSide, afterSeq uint32, wait time.Duration) (LinkMessages, error) {
	response, err := c.rpc.WaitMessages(ctx, connect.NewRequest(&syncv1.WaitMessagesRequest{
		ChannelId: channelID, Reader: writeLinkSide(reader), AfterSeq: afterSeq, WaitSeconds: uint32(max(wait, 0) / time.Second),
	}))
	if err != nil {
		return LinkMessages{}, readLinkError(err)
	}
	messages := make([]LinkMessage, 0, len(response.Msg.GetMessages()))
	for _, message := range response.Msg.GetMessages() {
		messages = append(messages, LinkMessage{Seq: message.GetSeq(), Body: message.GetBody()})
	}
	return LinkMessages{Messages: messages, IsClosed: response.Msg.GetIsClosed()}, nil
}

// CloseChannel ends the channel at once.
func (c *LinkClient) CloseChannel(ctx context.Context, channelID []byte) error {
	if _, err := c.rpc.CloseChannel(ctx, connect.NewRequest(&syncv1.CloseChannelRequest{ChannelId: channelID})); err != nil {
		return readLinkError(err)
	}
	return nil
}

func writeLinkSide(side LinkSide) syncv1.LinkSide {
	if side == LinkSideDisplayer {
		return syncv1.LinkSide_LINK_SIDE_DISPLAYER
	}
	if side == LinkSideScanner {
		return syncv1.LinkSide_LINK_SIDE_SCANNER
	}
	return syncv1.LinkSide_LINK_SIDE_UNSPECIFIED
}

func readLinkError(err error) error {
	failure, ok := errors.AsType[*connect.Error](err)
	if !ok {
		return syncprotocol.ReadError(err)
	}
	switch failure.Code() {
	case connect.CodeNotFound:
		return syncprotocol.NewLinkRefusal(syncprotocol.ErrChannelGone, failure)
	case connect.CodeAborted:
		return syncprotocol.NewLinkRefusal(syncprotocol.ErrLinkBusy, failure)
	case connect.CodeResourceExhausted:
		return syncprotocol.NewLinkRefusal(syncprotocol.ErrRateLimited, failure)
	}
	return syncprotocol.ReadError(err)
}

// AppendChain presents a longer chain and returns the chain the server stores after the call. A chain that does not extend
// the stored one is ChainStaleError with the stored head; a device that is not active in it is ErrWriteNotAllowed.
func (c *AccountClient) AppendChain(ctx context.Context, chain []byte) ([]byte, error) {
	response, err := c.rpc.AppendChain(ctx, connect.NewRequest(&syncv1.AppendChainRequest{Chain: chain}))
	if err != nil {
		return nil, syncprotocol.ReadCallError(err)
	}
	return response.Msg.GetChain(), nil
}
