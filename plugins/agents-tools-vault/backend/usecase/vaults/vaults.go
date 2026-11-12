// Package vaults describes the vaults of the installation and, when the host lists them, adds, renames,
// removes and opens them.
package vaults

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/vaultaddress"
)

// Vault is what the use case reads of one vault.
type Vault struct {
	Selector ports.VaultSelector
	Notes    ports.NoteLookup
}

// UseCase works the vaults of the installation for a call made in the current vault.
type UseCase struct {
	current domain.VaultID
	list    ports.VaultList
	open    func(domain.VaultID) Vault
}

// New creates a UseCase for a call made in the current vault. list is nil when the host gives no list
// of vaults. open gives the reads on the vault with an identity, and on the vault the host shows for the
// empty identity.
func New(current domain.VaultID, list ports.VaultList, open func(domain.VaultID) Vault) *UseCase {
	return &UseCase{current: current, list: list, open: open}
}

// Get returns the identifier, name, folder and note count of the vault named, or of the current vault.
func (u *UseCase) Get(ctx context.Context, in api.VaultInfoInput) (api.VaultInfoOutput, error) {
	vault := u.open(u.current)
	if in.Vault != "" {
		target, err := u.find(ctx, in.Vault)
		if err != nil {
			return api.VaultInfoOutput{}, err
		}
		vault = u.open(target.ID)
	}
	info, err := vault.Selector.GetVault(ctx)
	if err != nil {
		return api.VaultInfoOutput{}, err
	}
	noteCount := 0
	if paths, err := vault.Notes.GetPathsOfType(ctx, domain.TypeNote); err == nil {
		noteCount = len(paths)
	}
	return api.VaultInfoOutput{ID: string(info.ID), Name: info.Name, Path: info.Path, NoteCount: noteCount}, nil
}

// List returns every vault of the installation and marks the current one.
func (u *UseCase) List(ctx context.Context, _ api.VaultListInput) (api.VaultListOutput, error) {
	entries, err := u.getEntries(ctx)
	if err != nil {
		return api.VaultListOutput{}, err
	}
	out := api.VaultListOutput{Vaults: make([]api.VaultEntry, 0, len(entries))}
	for _, entry := range entries {
		out.Vaults = append(out.Vaults, api.VaultEntry{
			ID:        string(entry.ID),
			Name:      entry.Name,
			Path:      entry.Path,
			IsCurrent: entry.ID == u.current,
			IsMissing: entry.IsMissing,
		})
	}
	return out, nil
}

// Add adds a folder to the list of vaults.
func (u *UseCase) Add(ctx context.Context, in api.VaultAddInput) (api.VaultAddOutput, error) {
	vault, err := u.list.AddVault(ctx, in.Path, in.Name)
	if err != nil {
		return api.VaultAddOutput{}, err
	}
	return api.VaultAddOutput{ID: string(vault.ID), Name: vault.Name, Path: vault.Path}, nil
}

// Rename changes the display name of a vault.
func (u *UseCase) Rename(ctx context.Context, in api.VaultRenameInput) (api.VaultRenameOutput, error) {
	target, err := u.find(ctx, in.Vault)
	if err != nil {
		return api.VaultRenameOutput{}, err
	}
	result, err := u.list.RenameVault(ctx, target.ID, in.Name)
	if err != nil {
		return api.VaultRenameOutput{}, err
	}
	return api.VaultRenameOutput{ID: string(result.ID), Name: result.Name, Path: result.Path}, nil
}

// Remove removes a vault other than the current one from the list; its folder stays.
func (u *UseCase) Remove(ctx context.Context, in api.VaultRemoveInput) (api.VaultRemoveOutput, error) {
	target, err := u.findOtherVault(ctx, in.Vault)
	if err != nil {
		return api.VaultRemoveOutput{}, err
	}
	folder, err := u.list.RemoveVault(ctx, target.ID)
	if err != nil {
		return api.VaultRemoveOutput{}, err
	}
	return api.VaultRemoveOutput{Path: folder}, nil
}

// Open makes the window show a vault other than the current one.
func (u *UseCase) Open(ctx context.Context, in api.VaultOpenInput) (api.VaultOpenOutput, error) {
	target, err := u.findOtherVault(ctx, in.Vault)
	if err != nil {
		return api.VaultOpenOutput{}, err
	}
	if err := u.list.OpenVault(ctx, target.ID); err != nil {
		return api.VaultOpenOutput{}, err
	}
	return api.VaultOpenOutput{ID: string(target.ID), Name: target.Name, Path: target.Path}, nil
}

func (u *UseCase) getEntries(ctx context.Context) ([]host.VaultEntry, error) {
	if u.list != nil {
		return u.list.GetVaultList(ctx)
	}
	found, err := u.open("").Selector.GetVaults(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]host.VaultEntry, 0, len(found))
	for _, vault := range found {
		entries = append(entries, host.VaultEntry{Vault: vault, IsMissing: false})
	}
	return entries, nil
}

func (u *UseCase) findOtherVault(ctx context.Context, address string) (host.VaultEntry, error) {
	target, err := u.find(ctx, address)
	if err != nil {
		return host.VaultEntry{}, err
	}
	if target.ID == u.current {
		return host.VaultEntry{}, fmt.Errorf("%s is the current vault", address)
	}
	return target, nil
}

func (u *UseCase) find(ctx context.Context, address string) (host.VaultEntry, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return host.VaultEntry{}, errors.New("vault is required")
	}
	entries, err := u.getEntries(ctx)
	if err != nil {
		return host.VaultEntry{}, err
	}
	if entry, ok := vaultaddress.Find(entries, address); ok {
		return entry, nil
	}
	return host.VaultEntry{}, fmt.Errorf("unknown vault %q: no vault has this identifier, name or folder; call vault_list", address)
}
