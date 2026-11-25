package catalog

import (
	"context"
	"encoding/hex"
	"slices"

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
	return u.ExecuteHeld(ctx, remoteID, "")
}

// ExecuteHeld does what Execute does for a caller that holds the lock of the vault held already.
func (u *DeleteCloudVault) ExecuteHeld(ctx context.Context, remoteID string, held domain.VaultID) error {
	id, err := hex.DecodeString(remoteID)
	if err != nil {
		return ErrUnknownRemoteVault
	}
	linked, err := u.listLinkedVaults(ctx, remoteID)
	if err != nil {
		return err
	}
	unlock, err := u.lockLinked(ctx, linked, held)
	if err != nil {
		return err
	}
	defer unlock()
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

func (u *DeleteCloudVault) listLinkedVaults(ctx context.Context, remoteID string) ([]domain.VaultID, error) {
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
	slices.Sort(linked)
	return linked, nil
}

func (u *DeleteCloudVault) lockLinked(ctx context.Context, linked []domain.VaultID, held domain.VaultID) (func(), error) {
	var unlocks []func()
	unlockAll := func() {
		for _, unlock := range slices.Backward(unlocks) {
			unlock()
		}
	}
	for _, vault := range linked {
		if vault == held {
			continue
		}
		unlock, err := u.reader.deps.Locker.LockVault(ctx, vault)
		if err != nil {
			unlockAll()
			return nil, err
		}
		unlocks = append(unlocks, unlock)
	}
	return unlockAll, nil
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
