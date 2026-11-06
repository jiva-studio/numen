package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// VaultAddInput is the input of the vault_add tool.
type VaultAddInput struct {
	Path string `json:"path" jsonschema:"absolute path of the folder on this machine"`
	Name string `json:"name,omitempty" jsonschema:"display name of the vault; the folder's name when left out"`
}

// VaultAddOutput is the output of the vault_add tool.
type VaultAddOutput struct {
	IsAdded bool   `json:"added"`
	ID      string `json:"id" jsonschema:"identifier of the vault added"`
	Name    string `json:"name" jsonschema:"display name of the vault added"`
	Folder  string `json:"folder" jsonschema:"root directory of the vault added"`
}

// VaultRenameInput is the input of the vault_rename tool.
type VaultRenameInput struct {
	Vault string `json:"vault" jsonschema:"the vault's name, folder or identifier, as vault_list gives them"`
	Name  string `json:"name" jsonschema:"the new display name"`
}

// VaultRenameOutput is the output of the vault_rename tool.
type VaultRenameOutput struct {
	ID     string `json:"id" jsonschema:"identifier of the vault"`
	Name   string `json:"name" jsonschema:"display name of the vault after the rename"`
	Folder string `json:"folder" jsonschema:"root directory of the vault"`
}

// VaultRemoveInput is the input of the vault_remove tool.
type VaultRemoveInput struct {
	Vault string `json:"vault" jsonschema:"the vault's name, folder or identifier, as vault_list gives them"`
}

// VaultRemoveOutput is the output of the vault_remove tool.
type VaultRemoveOutput struct {
	IsRemoved bool   `json:"removed"`
	Folder    string `json:"folder" jsonschema:"root directory of the vault, which stays on the disk"`
}

// VaultOpenInput is the input of the vault_open tool.
type VaultOpenInput struct {
	Vault string `json:"vault" jsonschema:"the vault's name, folder or identifier, as vault_list gives them"`
}

// VaultOpenOutput is the output of the vault_open tool.
type VaultOpenOutput struct {
	IsOpening bool   `json:"opening"`
	Status    string `json:"status" jsonschema:"what happens to the window and to this session"`
}

func (t *Tools) addVaultAdd(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "vault_add",
		Title: "Add Vault",
		Description: "Add a folder of this machine to the installation's list of vaults, by its absolute path. " +
			"A folder inside a vault on the list, or one that holds one, is refused.",
		Effect:  tools.ReadWrite,
		Servers: []tools.Server{tools.Full},
	}, inVault(t, (*Tools).addVault))
}

func (t *Tools) addVaultRename(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.Move,
		Name:  "vault_rename",
		Title: "Rename Vault",
		Description: "Change the display name of a vault, named by its name, folder or identifier. A name " +
			"another vault has is refused.",
		Effect:  tools.ReadWrite,
		Servers: []tools.Server{tools.Full},
	}, inVault(t, (*Tools).renameVault))
}

func (t *Tools) addVaultRemove(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.Remove,
		Name:  "vault_remove",
		Title: "Remove Vault",
		Description: "Take a vault off the installation's list; its folder stays on the disk. The vault this " +
			"call works on and the last vault are refused.",
		Effect:  tools.ReadWrite,
		Servers: []tools.Server{tools.Full},
	}, inVault(t, (*Tools).removeVault))
}

func (t *Tools) addVaultOpen(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "vault_open",
		Title: "Open Vault",
		Description: "Make the person's window show another vault, named by its name, folder or identifier. " +
			"The window moves after the answer, and this session's view of the vault it works on ends. Call " +
			"it only when the person asks for it.",
		Effect:  tools.ReadWrite,
		Servers: []tools.Server{tools.Full},
	}, inVault(t, (*Tools).openVault))
}

func (t *Tools) addVault(ctx context.Context, in VaultAddInput) (VaultAddOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return VaultAddOutput{}, errors.New("path required")
	}
	vault, err := t.list.AddVault(ctx, in.Path, in.Name)
	if err != nil {
		return VaultAddOutput{}, fmt.Errorf("add vault %s: %w", in.Path, err)
	}
	return VaultAddOutput{IsAdded: true, ID: string(vault.ID), Name: vault.Name, Folder: vault.Path}, nil
}

func (t *Tools) renameVault(ctx context.Context, in VaultRenameInput) (VaultRenameOutput, error) {
	if strings.TrimSpace(in.Name) == "" {
		return VaultRenameOutput{}, errors.New("name required")
	}
	target, err := t.findVault(ctx, in.Vault)
	if err != nil {
		return VaultRenameOutput{}, err
	}
	result, err := t.list.RenameVault(ctx, target.ID, in.Name)
	if err != nil {
		return VaultRenameOutput{}, fmt.Errorf("rename vault %s: %w", in.Vault, err)
	}
	return VaultRenameOutput{ID: string(result.ID), Name: result.Name, Folder: result.Path}, nil
}

func (t *Tools) removeVault(ctx context.Context, in VaultRemoveInput) (VaultRemoveOutput, error) {
	target, err := t.findOtherVault(ctx, in.Vault, "remove")
	if err != nil {
		return VaultRemoveOutput{}, err
	}
	folder, err := t.list.RemoveVault(ctx, target.ID)
	if err != nil {
		return VaultRemoveOutput{}, fmt.Errorf("remove vault %s: %w", in.Vault, err)
	}
	return VaultRemoveOutput{IsRemoved: true, Folder: folder}, nil
}

func (t *Tools) openVault(ctx context.Context, in VaultOpenInput) (VaultOpenOutput, error) {
	target, err := t.findOtherVault(ctx, in.Vault, "open")
	if err != nil {
		return VaultOpenOutput{}, err
	}
	if err := t.list.OpenVault(ctx, target.ID); err != nil {
		return VaultOpenOutput{}, fmt.Errorf("open vault %s: %w", in.Vault, err)
	}
	status := fmt.Sprintf("The window is opening %s, and this session's view of the vault it works on ends.", target.Name)
	return VaultOpenOutput{IsOpening: true, Status: status}, nil
}

// findOtherVault finds the vault the address names, and refuses the vault the call works on.
func (t *Tools) findOtherVault(ctx context.Context, address, action string) (host.KnownVault, error) {
	target, err := t.findVault(ctx, address)
	if err != nil {
		return host.KnownVault{}, err
	}
	if target.ID == domain.VaultID(tools.GetVault(ctx)) {
		return host.KnownVault{}, fmt.Errorf("%s is the vault this call works on, and a call cannot %s it", address, action)
	}
	return target, nil
}

// findVault finds the vault of the list whose identifier, name or folder is the address, in that order.
func (t *Tools) findVault(ctx context.Context, address string) (host.KnownVault, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return host.KnownVault{}, errors.New("vault required")
	}
	vaults, err := t.list.GetVaultList(ctx)
	if err != nil {
		return host.KnownVault{}, fmt.Errorf("list vaults: %w", err)
	}
	for _, isMatch := range []func(host.KnownVault) bool{
		func(v host.KnownVault) bool { return string(v.ID) == address },
		func(v host.KnownVault) bool { return v.Name == address },
		func(v host.KnownVault) bool { return filepath.Clean(v.Path) == filepath.Clean(address) },
	} {
		for _, v := range vaults {
			if isMatch(v) {
				return v, nil
			}
		}
	}
	return host.KnownVault{}, fmt.Errorf("no vault has the identifier, name or folder %q; vault_list lists them: %w", address, host.ErrUnknownVault)
}
