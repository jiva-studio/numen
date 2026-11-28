package identity

import (
	"context"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// AccountLink is what a plugin with the identity permission may ask of the link in progress and of the chain the
// server holds, over the identity of an Account.
type AccountLink struct{ account *Account }

// NewAccountLink returns the link and chain calls over the identity of the account.
func NewAccountLink(account *Account) *AccountLink { return &AccountLink{account: account} }

// GetLinkWork returns what the courier of the link carries.
func (a *AccountLink) GetLinkWork(ctx context.Context, sinceVersion uint64, wait time.Duration) (identitydomain.LinkWork, error) {
	deps, err := a.account.open()
	if err != nil {
		return identitydomain.LinkWork{}, err
	}
	return NewGetLinkWork(deps).Execute(ctx, sinceVersion, wait)
}

// SetLinkChannel gives the link the channel the courier made.
func (a *AccountLink) SetLinkChannel(ctx context.Context, id []byte) error {
	deps, err := a.account.open()
	if err != nil {
		return err
	}
	return NewSetLinkChannel(deps).Execute(ctx, id)
}

// PutLinkInbound hands the core a blob of the other side.
func (a *AccountLink) PutLinkInbound(ctx context.Context, blob []byte) error {
	deps, err := a.account.open()
	if err != nil {
		return err
	}
	return NewPutLinkInbound(deps).Execute(ctx, blob)
}

// GetChainPublish returns the chain the server must hold before the core goes on.
func (a *AccountLink) GetChainPublish(ctx context.Context) (identitydomain.ChainPublish, error) {
	deps, err := a.account.open()
	if err != nil {
		return identitydomain.ChainPublish{}, err
	}
	return NewGetChainPublish(deps).Execute(ctx)
}

// PutChainPublished tells the core the answer of the server to the chain it published.
func (a *AccountLink) PutChainPublished(ctx context.Context, accepted bool, latest []byte) error {
	deps, err := a.account.open()
	if err != nil {
		return err
	}
	return NewPutChainPublished(deps).Execute(ctx, accepted, latest)
}

// AdoptChain gives the core a chain the server holds.
func (a *AccountLink) AdoptChain(ctx context.Context, chain []byte) error {
	deps, err := a.account.open()
	if err != nil {
		return err
	}
	return NewAdoptChain(deps).Execute(ctx, chain)
}
