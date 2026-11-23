package account

import (
	"context"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// LinkWork is what the courier of a link carries between the core of this device and the relay.
//
//numen:api
type LinkWork struct {
	// Version changes at each change of the link; pass it back to acknowledge the blobs it covers.
	Version uint64
	// IsActive is false when no link is in progress.
	IsActive bool
	// WantsChannel is true when the device that shows the QR code waits for a channel id from the relay.
	WantsChannel bool
	// ChannelID is the channel of the link, 16 bytes, once known.
	ChannelID []byte
	// Server is the origin the QR code names, empty for the default server.
	Server string
	// Outbound holds the blobs to post to the other side, in order.
	Outbound [][]byte
	// IsClosed is true when the link ended: close the channel.
	IsClosed bool
}

// ChainPublish is a chain the server must hold before the core goes on.
//
//numen:api
type ChainPublish struct {
	// IsPending is false when the core waits for no chain.
	IsPending bool
	// Chain is the chain to append to the account chain of the server.
	Chain []byte
}

// GetLinkWork returns what the courier carries. Where the version differs from sinceVersion it answers at once, and else
// after wait, at most 20 seconds, or the next change. It drops the blobs the version sinceVersion covers.
func (c *Client) GetLinkWork(ctx context.Context, sinceVersion uint64, wait time.Duration) (LinkWork, error) {
	response, err := c.rpc.GetLinkWork(ctx, connect.NewRequest(&v1.GetLinkWorkRequest{
		SinceVersion: sinceVersion, WaitSeconds: uint32(max(wait, 0) / time.Second),
	}))
	if err != nil {
		return LinkWork{}, readError(err)
	}
	return LinkWork{
		Version:      response.Msg.GetVersion(),
		IsActive:     response.Msg.GetIsActive(),
		WantsChannel: response.Msg.GetWantsChannel(),
		ChannelID:    response.Msg.GetChannelId(),
		Server:       response.Msg.GetServer(),
		Outbound:     response.Msg.GetOutbound(),
		IsClosed:     response.Msg.GetIsClosed(),
	}, nil
}

// SetLinkChannel gives the link the channel of 16 bytes the relay made. It returns ErrNoLink where no link waits for one.
func (c *Client) SetLinkChannel(ctx context.Context, id []byte) error {
	if _, err := c.rpc.SetLinkChannel(ctx, connect.NewRequest(&v1.SetLinkChannelRequest{ChannelId: id})); err != nil {
		return readError(err)
	}
	return nil
}

// PutLinkInbound hands the core a blob the other side posted. It returns ErrNoLink where no link is in progress.
func (c *Client) PutLinkInbound(ctx context.Context, blob []byte) error {
	if _, err := c.rpc.PutLinkInbound(ctx, connect.NewRequest(&v1.PutLinkInboundRequest{Blob: blob})); err != nil {
		return readError(err)
	}
	return nil
}

// GetChainPublish returns the chain the core waits to have accepted by the server.
func (c *Client) GetChainPublish(ctx context.Context) (ChainPublish, error) {
	response, err := c.rpc.GetChainPublish(ctx, connect.NewRequest(&v1.GetChainPublishRequest{}))
	if err != nil {
		return ChainPublish{}, readError(err)
	}
	return ChainPublish{IsPending: response.Msg.GetIsPending(), Chain: response.Msg.GetChain()}, nil
}

// PutChainPublished tells the core the answer of the server to the chain it published. Where another device appended
// first, accepted is false and latest is the chain the server holds. It returns ErrNoLink where nothing waited for the server.
func (c *Client) PutChainPublished(ctx context.Context, accepted bool, latest []byte) error {
	if _, err := c.rpc.PutChainPublished(ctx, connect.NewRequest(&v1.PutChainPublishedRequest{Accepted: accepted, Latest: latest})); err != nil {
		return readError(err)
	}
	return nil
}

// AdoptChain gives the core a chain the server holds. It returns ErrInvalidChain for bytes that are not a chain or that
// do not extend the chain of this device.
func (c *Client) AdoptChain(ctx context.Context, chain []byte) error {
	if _, err := c.rpc.AdoptChain(ctx, connect.NewRequest(&v1.AdoptChainRequest{Chain: chain})); err != nil {
		return readError(err)
	}
	return nil
}
