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
	ID   string `json:"id" jsonschema:"identifier of the new vault"`
	Name string `json:"name" jsonschema:"display name of the new vault"`
	Path string `json:"path" jsonschema:"root directory of the new vault"`
}

// VaultRenameInput is the input of the vault_rename tool.
type VaultRenameInput struct {
	Vault string `json:"vault" jsonschema:"the vault's name, path or identifier, as vault_list returns them"`
	Name  string `json:"name" jsonschema:"the new display name"`
}

// VaultRenameOutput is the output of the vault_rename tool.
type VaultRenameOutput struct {
	ID   string `json:"id" jsonschema:"identifier of the vault"`
	Name string `json:"name" jsonschema:"display name of the vault after the rename"`
	Path string `json:"path" jsonschema:"root directory of the vault"`
}

// VaultRemoveInput is the input of the vault_remove tool.
type VaultRemoveInput struct {
	Vault string `json:"vault" jsonschema:"the vault's name, path or identifier, as vault_list returns them"`
}

// VaultRemoveOutput is the output of the vault_remove tool.
type VaultRemoveOutput struct {
	Path string `json:"path" jsonschema:"root directory of the vault, which is not deleted"`
}

// VaultOpenInput is the input of the vault_open tool.
type VaultOpenInput struct {
	Vault string `json:"vault" jsonschema:"the vault's name, path or identifier, as vault_list returns them"`
}

// VaultOpenOutput is the output of the vault_open tool.
type VaultOpenOutput struct {
	ID   string `json:"id" jsonschema:"identifier of the vault the window opens"`
	Name string `json:"name" jsonschema:"display name of the vault the window opens"`
	Path string `json:"path" jsonschema:"root directory of the vault the window opens"`
}

func (t *Tools) addVaultAdd(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "vault_add",
		Title: "Add Vault",
		Description: "Add a folder of this machine to the installation's list of vaults, by its absolute path. " +
			"A folder that is inside a vault on the list, or that contains one, is rejected.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{vaultAddSentence},
		Requires:     []string{host.PermissionVaultManage},
	}, inVault(t, (*Tools).addVault))
}

func (t *Tools) addVaultRename(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.Move,
		Name:  "vault_rename",
		Title: "Rename Vault",
		Description: "Change the display name of a vault, named by its name, path or identifier. A name " +
			"another vault has is rejected.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{vaultRenameSentence},
		Requires:     []string{host.PermissionVaultManage},
	}, inVault(t, (*Tools).renameVault))
}

func (t *Tools) addVaultRemove(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.Remove,
		Name:  "vault_remove",
		Title: "Remove Vault",
		Description: "Remove a vault from the installation's list; its folder is not deleted. The current " +
			"vault and the last vault are rejected.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{vaultRemoveSentence},
		Requires:     []string{host.PermissionVaultManage},
	}, inVault(t, (*Tools).removeVault))
}

func (t *Tools) addVaultOpen(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "vault_open",
		Title: "Open Vault",
		Description: "Switch the window to another vault, named by its name, path or identifier, and return " +
			"when the window shows it; this session then loses access to the current vault. If the window " +
			"has unsaved changes, the vault is not switched, the window prompts the user, and the call returns " +
			"an error; call it again after the user responds. Call it only when the user asks for it.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{vaultOpenSentence, vaultOpenPromptSentence},
		Requires:     []string{host.PermissionVaultManage},
	}, inVault(t, (*Tools).openVault))
}

func (t *Tools) addVault(ctx context.Context, in VaultAddInput) (VaultAddOutput, error) {
	vault, err := t.list.AddVault(ctx, in.Path, in.Name)
	if err != nil {
		return VaultAddOutput{}, err
	}
	return VaultAddOutput{ID: string(vault.ID), Name: vault.Name, Path: vault.Path}, nil
}

func (t *Tools) renameVault(ctx context.Context, in VaultRenameInput) (VaultRenameOutput, error) {
	target, err := t.findVault(ctx, in.Vault)
	if err != nil {
		return VaultRenameOutput{}, err
	}
	result, err := t.list.RenameVault(ctx, target.ID, in.Name)
	if err != nil {
		return VaultRenameOutput{}, err
	}
	return VaultRenameOutput{ID: string(result.ID), Name: result.Name, Path: result.Path}, nil
}

func (t *Tools) removeVault(ctx context.Context, in VaultRemoveInput) (VaultRemoveOutput, error) {
	target, err := t.findOtherVault(ctx, in.Vault)
	if err != nil {
		return VaultRemoveOutput{}, err
	}
	folder, err := t.list.RemoveVault(ctx, target.ID)
	if err != nil {
		return VaultRemoveOutput{}, err
	}
	return VaultRemoveOutput{Path: folder}, nil
}

func (t *Tools) openVault(ctx context.Context, in VaultOpenInput) (VaultOpenOutput, error) {
	target, err := t.findOtherVault(ctx, in.Vault)
	if err != nil {
		return VaultOpenOutput{}, err
	}
	if err := t.list.OpenVault(ctx, target.ID); err != nil {
		return VaultOpenOutput{}, err
	}
	return VaultOpenOutput{ID: string(target.ID), Name: target.Name, Path: target.Path}, nil
}

// findOtherVault finds the vault the address names, and rejects the current vault.
func (t *Tools) findOtherVault(ctx context.Context, address string) (host.VaultEntry, error) {
	target, err := t.findVault(ctx, address)
	if err != nil {
		return host.VaultEntry{}, err
	}
	if target.ID == domain.VaultID(tools.GetVault(ctx)) {
		return host.VaultEntry{}, fmt.Errorf("%s is the current vault", address)
	}
	return target, nil
}

// findVault finds the vault of the list whose identifier, name or folder is the address, in that order.
func (t *Tools) findVault(ctx context.Context, address string) (host.VaultEntry, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return host.VaultEntry{}, errors.New("vault is required")
	}
	vaults, err := t.getVaultEntries(ctx)
	if err != nil {
		return host.VaultEntry{}, err
	}
	for _, isMatch := range []func(host.VaultEntry) bool{
		func(v host.VaultEntry) bool { return string(v.ID) == address },
		func(v host.VaultEntry) bool { return v.Name == address },
		func(v host.VaultEntry) bool { return filepath.Clean(v.Path) == filepath.Clean(address) },
	} {
		for _, v := range vaults {
			if isMatch(v) {
				return v, nil
			}
		}
	}
	return host.VaultEntry{}, fmt.Errorf("unknown vault %q: no vault has this identifier, name or folder; call vault_list", address)
}
