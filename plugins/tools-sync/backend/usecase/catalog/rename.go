package catalog

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

// RenameRemoteVault gives the synced vault the name of its local vault.
type RenameRemoteVault struct{ reader reader }

// NewRenameRemoteVault returns the use case over the dependencies.
func NewRenameRemoteVault(deps Deps) *RenameRemoteVault {
	return &RenameRemoteVault{reader: reader{deps: deps}}
}

// Execute appends an entry with the name of the local vault when it differs from the one the link holds, and
// reports whether it did.
func (u *RenameRemoteVault) Execute(ctx context.Context, id domain.VaultID) (bool, error) {
	link, err := u.reader.deps.State.ReadLink(ctx, id)
	if err != nil {
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
	key, err := u.reader.deps.Crypto.OpenData(ctx, vaultlink.GetStateKeyContext(remoteID), link.SealedKey)
	if err != nil {
		return fmt.Errorf("open the vault key: %w", err)
	}
	keys, err := u.readKeys(ctx, remoteID)
	if err != nil {
		return err
	}
	body := vaultlink.CatalogBody{LocalID: string(vault.ID), Name: vault.Name, Key: key, Keys: keys}
	if err := u.reader.appendEntry(ctx, entryInput{vaultID: remoteID, isCreating: false, body: body}); err != nil {
		return err
	}
	link.CatalogName = vault.Name
	return u.reader.deps.State.WriteLink(ctx, vault.ID, link)
}

// readKeys returns the keys by epoch the last entry of the vault lists, so a new name does not drop the keys a revoke gave it.
func (u *RenameRemoteVault) readKeys(ctx context.Context, remoteID []byte) ([]vaultlink.VaultKey, error) {
	entries, err := u.reader.read(ctx)
	if err != nil {
		return nil, err
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if bytes.Equal(entries[i].GetVaultId(), remoteID) {
			body, err := u.reader.openBody(ctx, entries[i])
			return body.Keys, err
		}
	}
	return nil, nil
}
