package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

// VaultInfoInput is the input of the vault_get tool.
type VaultInfoInput struct {
	Vault string `json:"vault,omitempty" jsonschema:"the vault's name, path or identifier, as vault_list returns them; the vault of the call when left out"`
}

// VaultInfoOutput is the output of the vault_get tool.
type VaultInfoOutput struct {
	ID        string `json:"id" jsonschema:"vault unique identifier"`
	Name      string `json:"name" jsonschema:"vault display name"`
	Path      string `json:"path" jsonschema:"vault root directory path on disk"`
	NoteCount int    `json:"noteCount" jsonschema:"total number of indexed notes in the vault"`
}

func (t *Tools) addVaultGet(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:        tools.ByEffect,
		Name:        "vault_get",
		Title:       "Get Vault Info",
		Description: "Get the identifier, name, root path and note count of a vault, named by its name, path or identifier; the vault of the call when none is named.",
		Effect:      tools.ReadOnly,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).getVaultInfo))
}

func (t *Tools) getVaultInfo(ctx context.Context, in VaultInfoInput) (VaultInfoOutput, error) {
	v := t.vault
	if in.Vault != "" {
		target, err := t.findVault(ctx, in.Vault)
		if err != nil {
			return VaultInfoOutput{}, err
		}
		v = t.host.With(target.ID)
	}

	info, err := v.GetVault(ctx)
	if err != nil {
		return VaultInfoOutput{}, err
	}

	noteCount := 0
	paths, err := v.GetPathsOfType(ctx, domain.TypeNote)
	if err == nil {
		noteCount = len(paths)
	}

	return VaultInfoOutput{
		ID:        string(info.ID),
		Name:      info.Name,
		Path:      info.Path,
		NoteCount: noteCount,
	}, nil
}
