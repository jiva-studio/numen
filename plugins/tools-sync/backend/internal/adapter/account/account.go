// Package account reads and signs with the identity of this device through the host.
package account

import (
	"context"
	"errors"

	sdk "github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
)

// Account is the identity of this device, asked of the host.
type Account struct {
	client *sdk.Client
}

// New returns the identity served by the client.
func New(client *sdk.Client) *Account { return &Account{client: client} }

// GetPublicIdentity returns what is public of the identity, or session.ErrNoIdentity.
func (a *Account) GetPublicIdentity(ctx context.Context) (session.Identity, error) {
	public, err := a.client.GetPublicIdentity(ctx)
	if err != nil {
		return session.Identity{}, readError(err)
	}
	return session.Identity{
		AccountFingerprint: public.AccountFingerprint, DeviceSignKey: public.DeviceSignKey, Chain: public.Chain,
		KeyEpoch: public.KeyEpoch, ChainLength: public.ChainLength, IsRevoked: public.IsRevoked,
	}, nil
}

// SignAsDevice signs the message with the key of the device, or fails with session.ErrNoIdentity.
func (a *Account) SignAsDevice(ctx context.Context, message []byte) ([]byte, error) {
	signature, err := a.client.SignAsDevice(ctx, message)
	return signature, readError(err)
}

func readError(err error) error {
	switch {
	case errors.Is(err, sdk.ErrNoIdentity):
		return session.ErrNoIdentity
	case errors.Is(err, sdk.ErrCannotOpen):
		return session.ErrCannotOpen
	case errors.Is(err, sdk.ErrNoPendingRestore):
		return session.ErrNoPendingRestore
	case errors.Is(err, sdk.ErrInvalidChain):
		return session.ErrInvalidChain
	}
	return err
}

// SealData seals the plain bytes for every device of the account, bound to the context.
func (a *Account) SealData(ctx context.Context, dataContext, plain []byte) ([]byte, error) {
	sealed, err := a.client.SealData(ctx, dataContext, plain)
	return sealed, readError(err)
}

// OpenData opens what SealData sealed, or fails with session.ErrCannotOpen.
func (a *Account) OpenData(ctx context.Context, dataContext, sealed []byte) ([]byte, error) {
	plain, err := a.client.OpenData(ctx, dataContext, sealed)
	return plain, readError(err)
}

// GetPendingRestore returns the restore that waits for an account chain.
func (a *Account) GetPendingRestore(ctx context.Context) (session.PendingRestore, error) {
	pending, err := a.client.GetPendingRestore(ctx)
	if err != nil {
		return session.PendingRestore{}, readError(err)
	}
	return session.PendingRestore{IsPending: pending.IsPending, RootSignKey: pending.RootSignKey, DeviceSignKey: pending.DeviceSignKey}, nil
}

// SignAsRecovery signs the message with the recovery key of the pending restore.
func (a *Account) SignAsRecovery(ctx context.Context, message []byte) ([]byte, error) {
	signature, err := a.client.SignAsRecovery(ctx, message)
	return signature, readError(err)
}

// CompleteRestore finishes the pending restore with the chain, or with none.
func (a *Account) CompleteRestore(ctx context.Context, chain []byte) error {
	return readError(a.client.CompleteRestore(ctx, chain))
}

// OpenBox opens what was sealed to the account box key of this device for this plugin and the context, or fails with
// session.ErrCannotOpen.
func (a *Account) OpenBox(ctx context.Context, boxContext, sealed []byte) ([]byte, error) {
	plain, err := a.client.OpenSealedBox(ctx, boxContext, sealed)
	return plain, readError(err)
}
