package identity

import (
	"context"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// GetLinkWork returns what the courier of the link carries.
func (a *Account) GetLinkWork(ctx context.Context, sinceVersion uint64, wait time.Duration) (identitydomain.LinkWork, error) {
	deps, err := a.open()
	if err != nil {
		return identitydomain.LinkWork{}, err
	}
	return NewGetLinkWork(deps).Execute(ctx, sinceVersion, wait)
}

// SetLinkChannel gives the link the channel the courier made.
func (a *Account) SetLinkChannel(ctx context.Context, id []byte) error {
	deps, err := a.open()
	if err != nil {
		return err
	}
	return NewSetLinkChannel(deps).Execute(ctx, id)
}

// PutLinkInbound hands the core a blob of the other side.
func (a *Account) PutLinkInbound(ctx context.Context, blob []byte) error {
	deps, err := a.open()
	if err != nil {
		return err
	}
	return NewPutLinkInbound(deps).Execute(ctx, blob)
}

// GetChainPublish returns the chain the server must hold before the core goes on.
func (a *Account) GetChainPublish(ctx context.Context) (identitydomain.ChainPublish, error) {
	deps, err := a.open()
	if err != nil {
		return identitydomain.ChainPublish{}, err
	}
	return NewGetChainPublish(deps).Execute(ctx)
}

// PutChainPublished tells the core the answer of the server to the chain it published.
func (a *Account) PutChainPublished(ctx context.Context, accepted bool, latest []byte) error {
	deps, err := a.open()
	if err != nil {
		return err
	}
	return NewPutChainPublished(deps).Execute(ctx, accepted, latest)
}

// AdoptChain gives the core a chain the server holds.
func (a *Account) AdoptChain(ctx context.Context, chain []byte) error {
	deps, err := a.open()
	if err != nil {
		return err
	}
	return NewAdoptChain(deps).Execute(ctx, chain)
}
