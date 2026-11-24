package catalog

import (
	"context"
	"encoding/hex"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

// ShareCloser closes the shares of a vault of this device.
type ShareCloser interface {
	Execute(ctx context.Context, id domain.VaultID) error
}

// DeleteCloudVault deletes a synced vault of the account from the cloud.
type DeleteCloudVault struct {
	reader reader
	shares ShareCloser
}

// NewDeleteCloudVault returns the use case over the dependencies.
func NewDeleteCloudVault(deps Deps, shares ShareCloser) *DeleteCloudVault {
	return &DeleteCloudVault{reader: reader{deps: deps}, shares: shares}
}

// Execute closes the shares of the vault on this device, appends the entry that tells the other devices the vault is
// removed, deletes the vault on the server and forgets the links of this device to it. A vault the catalog does not
// know, or has removed, is deleted on the server all the same. The files of the local vaults stay.
func (u *DeleteCloudVault) Execute(ctx context.Context, remoteID string) error {
	id, err := hex.DecodeString(remoteID)
	if err != nil {
		return ErrUnknownRemoteVault
	}
	linked, err := u.readLinked(ctx, remoteID)
	if err != nil {
		return err
	}
	for _, vault := range linked {
		if err := u.shares.Execute(ctx, vault); err != nil {
			return err
		}
	}
	if err := u.markRemoved(ctx, id); err != nil {
		return err
	}
	if _, err := u.reader.deps.Cloud.DeleteVault(ctx, id); err != nil {
		return err
	}
	for _, vault := range linked {
		if err := u.reader.deps.Forget.ForgetVault(ctx, vault); err != nil {
			return err
		}
	}
	return nil
}

func (u *DeleteCloudVault) readLinked(ctx context.Context, remoteID string) ([]domain.VaultID, error) {
	links, err := u.reader.readLinks(ctx)
	if err != nil {
		return nil, err
	}
	var linked []domain.VaultID
	for vault, link := range links {
		if link.RemoteVaultID == remoteID {
			linked = append(linked, vault)
		}
	}
	return linked, nil
}

func (u *DeleteCloudVault) markRemoved(ctx context.Context, id []byte) error {
	entries, err := u.reader.read(ctx)
	if err != nil {
		return err
	}
	last := getLastEntry(entries, id)
	if last == nil {
		return nil
	}
	body, err := u.reader.openBody(ctx, last)
	if err != nil || body.Removed {
		return err
	}
	removed := vaultlink.CatalogBody{LocalID: "", Name: "", Key: nil, Keys: nil, Removed: true}
	return u.reader.appendEntry(ctx, entryInput{vaultID: id, isCreating: false, body: removed})
}
