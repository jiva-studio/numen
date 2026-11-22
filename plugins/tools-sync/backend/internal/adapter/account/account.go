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
	return session.Identity{AccountFingerprint: public.AccountFingerprint, DeviceSignKey: public.DeviceSignKey, Chain: public.Chain}, nil
}

// SignAsDevice signs the message with the key of the device, or fails with session.ErrNoIdentity.
func (a *Account) SignAsDevice(ctx context.Context, message []byte) ([]byte, error) {
	signature, err := a.client.SignAsDevice(ctx, message)
	return signature, readError(err)
}

func readError(err error) error {
	if errors.Is(err, sdk.ErrNoIdentity) {
		return session.ErrNoIdentity
	}
	return err
}
