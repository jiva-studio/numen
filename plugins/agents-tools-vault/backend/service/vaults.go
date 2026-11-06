package service

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// ListedVault is a vault of the installation.
type ListedVault struct {
	ID        string `json:"id" jsonschema:"vault identifier, which stays the same when its folder moves"`
	Name      string `json:"name" jsonschema:"vault display name"`
	Path      string `json:"path" jsonschema:"vault root directory on this machine"`
	IsCurrent bool   `json:"isCurrent" jsonschema:"true for the vault this call works on"`
}

// VaultListInput is the input of the vault_list tool.
type VaultListInput struct{}

// VaultListOutput is the output of the vault_list tool.
type VaultListOutput struct {
	Vaults []ListedVault `json:"vaults"`
}

// VaultProblemsInput is the input of the vault_problems tool.
type VaultProblemsInput struct {
	Checks []string `json:"checks,omitempty" jsonschema:"the checks to run; only parse is served, and it runs when left out"`
}

// VaultProblemItem is a problem found in one file of the vault.
type VaultProblemItem struct {
	Path   string `json:"path" jsonschema:"path of the file with the problem"`
	Check  string `json:"check" jsonschema:"the check that found the problem"`
	Detail string `json:"detail" jsonschema:"description of the problem in the terms of the file"`
}

// VaultProblemsOutput is the output of the vault_problems tool.
type VaultProblemsOutput struct {
	Problems []VaultProblemItem `json:"problems"`
	Checks   []string           `json:"checks" jsonschema:"the checks this result covers"`
}

func (t *Tools) addVaultList(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:        tools.ByEffect,
		Name:        "vault_list",
		Title:       "List Vaults",
		Description: "List every vault of the installation with its identifier, name and folder path. The vault this call works on has isCurrent set.",
		Effect:      tools.ReadOnly,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).listVaults))
}

func (t *Tools) addVaultProblems(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:        tools.ByEffect,
		Name:        "vault_problems",
		Title:       "List Vault Problems",
		Description: "List the files of the vault that could not be parsed, with the path and a description of each problem.",
		Effect:      tools.ReadOnly,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).listVaultProblems))
}

func (t *Tools) listVaults(ctx context.Context, _ VaultListInput) (VaultListOutput, error) {
	vaults, err := t.host.GetVaults(ctx)
	if err != nil {
		return VaultListOutput{}, fmt.Errorf("list vaults: %w", err)
	}
	current := domain.VaultID(tools.GetVault(ctx))
	out := VaultListOutput{Vaults: make([]ListedVault, 0, len(vaults))}
	for _, v := range vaults {
		out.Vaults = append(out.Vaults, ListedVault{
			ID:        string(v.ID),
			Name:      v.Name,
			Path:      v.Path,
			IsCurrent: v.ID == current,
		})
	}
	return out, nil
}

func (t *Tools) listVaultProblems(ctx context.Context, in VaultProblemsInput) (VaultProblemsOutput, error) {
	for _, check := range in.Checks {
		if check != string(domain.CheckParse) {
			return VaultProblemsOutput{}, fmt.Errorf("unknown check %q: only %s is served", check, domain.CheckParse)
		}
	}
	problems, err := t.vault.GetParseProblems(ctx)
	if err != nil {
		return VaultProblemsOutput{}, fmt.Errorf("get parse problems: %w", err)
	}
	out := VaultProblemsOutput{
		Problems: make([]VaultProblemItem, 0, len(problems)),
		Checks:   []string{string(domain.CheckParse)},
	}
	for _, p := range problems {
		out.Problems = append(out.Problems, VaultProblemItem{Path: p.Path, Check: string(p.Kind), Detail: p.Detail})
	}
	return out, nil
}
