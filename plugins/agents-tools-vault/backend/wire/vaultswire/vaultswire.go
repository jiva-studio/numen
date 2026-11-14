// Package vaultswire calls the vaults use case with the messages of the vault tools.
package vaultswire

import (
	"context"

	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/vaults"
)

// GetVault answers vault_get.
func GetVault(ctx context.Context, u *vaults.UseCase, in api.VaultInfoInput) (api.VaultInfoOutput, error) {
	info, err := u.Get(ctx, in.Vault)
	if err != nil {
		return api.VaultInfoOutput{}, err
	}
	return api.VaultInfoOutput{ID: string(info.ID), Name: info.Name, Path: info.Path, NoteCount: info.NoteCount}, nil
}

// ListVaults answers vault_list.
func ListVaults(ctx context.Context, u *vaults.UseCase, _ api.VaultListInput) (api.VaultListOutput, error) {
	entries, err := u.List(ctx)
	if err != nil {
		return api.VaultListOutput{}, err
	}
	out := api.VaultListOutput{Vaults: make([]api.VaultEntry, 0, len(entries))}
	for _, entry := range entries {
		out.Vaults = append(out.Vaults, api.VaultEntry{
			ID:        string(entry.ID),
			Name:      entry.Name,
			Path:      entry.Path,
			IsCurrent: entry.IsCurrent,
			IsMissing: entry.IsMissing,
		})
	}
	return out, nil
}

// AddVault answers vault_add.
func AddVault(ctx context.Context, u *vaults.UseCase, in api.VaultAddInput) (api.VaultAddOutput, error) {
	vault, err := u.Add(ctx, in.Path, in.Name)
	if err != nil {
		return api.VaultAddOutput{}, err
	}
	return api.VaultAddOutput{ID: string(vault.ID), Name: vault.Name, Path: vault.Path}, nil
}

// RenameVault answers vault_rename.
func RenameVault(ctx context.Context, u *vaults.UseCase, in api.VaultRenameInput) (api.VaultRenameOutput, error) {
	vault, err := u.Rename(ctx, in.Vault, in.Name)
	if err != nil {
		return api.VaultRenameOutput{}, err
	}
	return api.VaultRenameOutput{ID: string(vault.ID), Name: vault.Name, Path: vault.Path}, nil
}

// RemoveVault answers vault_remove.
func RemoveVault(ctx context.Context, u *vaults.UseCase, in api.VaultRemoveInput) (api.VaultRemoveOutput, error) {
	folder, err := u.Remove(ctx, in.Vault)
	if err != nil {
		return api.VaultRemoveOutput{}, err
	}
	return api.VaultRemoveOutput{Path: folder}, nil
}

// OpenVault answers vault_open.
func OpenVault(ctx context.Context, u *vaults.UseCase, in api.VaultOpenInput) (api.VaultOpenOutput, error) {
	vault, err := u.Open(ctx, in.Vault)
	if err != nil {
		return api.VaultOpenOutput{}, err
	}
	return api.VaultOpenOutput{ID: string(vault.ID), Name: vault.Name, Path: vault.Path}, nil
}
