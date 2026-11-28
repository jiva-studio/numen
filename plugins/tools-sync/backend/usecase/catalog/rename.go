package catalog

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

// RenameRemoteVault gives the synced vault the name of its local vault.
type RenameRemoteVault struct{ reader reader }

// NewRenameRemoteVault returns the use case over the dependencies.
func NewRenameRemoteVault(deps Deps) *RenameRemoteVault {
	return &RenameRemoteVault{reader: reader{deps: deps}}
}

// Execute appends an entry with the name of the local vault when it differs from the one the link holds, and
// reports whether it did. It forgets the link of a vault whose cloud copy another device deleted, and returns
// ErrNotConnected for it.
func (u *RenameRemoteVault) Execute(ctx context.Context, id domain.VaultID) (bool, error) {
	link, err := u.reader.deps.State.ReadLink(ctx, id)
	if err != nil {
		return false, err
	}
	if err := u.dropRemoved(ctx, id, link); err != nil {
		return false, err
	}
	vaults, err := u.reader.deps.Host.GetVaults(ctx)
	if err != nil {
		return false, err
	}
	for _, vault := range vaults {
		if vault.ID == id && vault.Name != link.CatalogName {
			return true, u.rename(ctx, vault, link)
		}
	}
	return false, nil
}

func (u *RenameRemoteVault) rename(ctx context.Context, vault domain.Vault, link vaultlink.Link) error {
	remoteID, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return fmt.Errorf("read the remote id of the link: %w", err)
	}
	opened, err := vaultlink.OpenKeys(ctx, u.reader.deps.Crypto, link)
	if err != nil {
		return err
	}
	keys, err := u.readKeys(ctx, remoteID)
	if err != nil {
		return err
	}
	body := vaultlink.CatalogBody{LocalID: string(vault.ID), Name: vault.Name, Key: opened[0].Key, Keys: keys, Removed: false}
	if err := u.reader.appendEntry(ctx, entryInput{vaultID: remoteID, isCreating: false, body: body}); err != nil {
		return err
	}
	return u.reader.deps.State.UpdateLink(ctx, vault.ID, func(fresh *vaultlink.Link) error {
		fresh.CatalogName = vault.Name
		return nil
	})
}

func (u *RenameRemoteVault) readKeys(ctx context.Context, remoteID []byte) ([]vaultlink.VaultKey, error) {
	entries, err := u.reader.read(ctx)
	if err != nil {
		return nil, err
	}
	if last := getLastEntry(entries, remoteID); last != nil {
		body, err := u.reader.openBody(ctx, last)
		return body.Keys, err
	}
	return nil, nil
}

func (u *RenameRemoteVault) dropRemoved(ctx context.Context, id domain.VaultID, link vaultlink.Link) error {
	remoteID, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return fmt.Errorf("read the remote id of the link: %w", err)
	}
	entries, err := u.reader.read(ctx)
	if err != nil {
		return err
	}
	last := getLastEntry(entries, remoteID)
	if last == nil {
		return nil
	}
	body, err := u.reader.openBody(ctx, last)
	if err != nil || !body.Removed {
		return err
	}
	if err := u.reader.deps.Forget.ForgetVault(ctx, id); err != nil {
		return err
	}
	return vaultlink.ErrNotConnected
}

func getLastEntry(entries []*syncv1.CatalogEntry, remoteID []byte) *syncv1.CatalogEntry {
	for _, entry := range slices.Backward(entries) {
		if bytes.Equal(entry.GetVaultId(), remoteID) {
			return entry
		}
	}
	return nil
}
