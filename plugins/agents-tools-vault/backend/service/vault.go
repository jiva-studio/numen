package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

type VaultInfoInput struct {
	ID string `json:"id,omitempty" jsonschema:"optional vault ID to query; defaults to the vault the call works"`
}

type VaultInfoOutput struct {
	ID        string `json:"id" jsonschema:"vault unique identifier"`
	Name      string `json:"name" jsonschema:"vault display name"`
	Path      string `json:"path" jsonschema:"vault root directory path on disk"`
	NoteCount int    `json:"note_count" jsonschema:"total number of indexed notes in the vault"`
}

func (t *Tools) addVaultInfo(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "vault_info",
		Title:       "Vault Info",
		Description: "Get metadata, name, root path, and note count for the vault.",
		Effect:      tools.ReadOnly,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).getVaultInfo))
}

func (t *Tools) addVaultGet(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "vault_get",
		Title:       "Get Vault Info",
		Description: "Get metadata, name, root path, and note count for the vault.",
		Effect:      tools.ReadOnly,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).getVaultInfo))
}

func (t *Tools) getVaultInfo(ctx context.Context, in VaultInfoInput) (VaultInfoOutput, error) {
	v := t.vault
	if in.ID != "" {
		v = t.host.With(domain.VaultID(in.ID))
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
