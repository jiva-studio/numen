package account

import (
	"context"
	"errors"
	"time"
)

// LinkWork is what the courier of a link carries between the core of this device and the relay.
//
//numen:api
type LinkWork struct {
	Version      uint64
	IsActive     bool
	WantsChannel bool
	ChannelID    []byte
	Server       string
	Outbound     [][]byte
	IsClosed     bool
}

// ChainPublish is a chain the server must hold before the core goes on.
//
//numen:api
type ChainPublish struct {
	IsPending bool
	Chain     []byte
}

func (c *Client) GetLinkWork(ctx context.Context, sinceVersion uint64, wait time.Duration) (LinkWork, error) {
	return LinkWork{}, errors.New("not implemented")
}

func (c *Client) SetLinkChannel(ctx context.Context, id []byte) error {
	return errors.New("not implemented")
}

func (c *Client) PutLinkInbound(ctx context.Context, blob []byte) error {
	return errors.New("not implemented")
}

func (c *Client) GetChainPublish(ctx context.Context) (ChainPublish, error) {
	return ChainPublish{}, errors.New("not implemented")
}

func (c *Client) PutChainPublished(ctx context.Context, accepted bool, latest []byte) error {
	return errors.New("not implemented")
}

func (c *Client) AdoptChain(ctx context.Context, chain []byte) error {
	return errors.New("not implemented")
}
