package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// getCheckNames returns the checks vault_problems runs. dangling runs only when the call names it.
func getCheckNames() []string {
	return []string{string(domain.CheckParse), "frontmatter", "ambiguous", "dangling"}
}

// VaultEntry is a vault of the installation.
type VaultEntry struct {
	ID        string `json:"id" jsonschema:"vault identifier, which stays the same when its folder moves"`
	Name      string `json:"name" jsonschema:"vault display name"`
	Path      string `json:"path" jsonschema:"vault root directory on this machine"`
	IsCurrent bool   `json:"isCurrent" jsonschema:"true for the vault this call works on"`
	IsMissing bool   `json:"isMissing,omitempty" jsonschema:"true when nothing is at the vault's folder"`
}

// VaultListInput is the input of the vault_list tool.
type VaultListInput struct{}

// VaultListOutput is the output of the vault_list tool.
type VaultListOutput struct {
	Vaults []VaultEntry `json:"vaults"`
}

// VaultProblemsInput is the input of the vault_problems tool.
type VaultProblemsInput struct {
	Checks []string `json:"checks,omitempty" jsonschema:"the checks to run: parse, frontmatter, ambiguous and dangling; parse, frontmatter and ambiguous run when left out"`
}

// VaultProblemItem is a problem found in one file of the vault.
type VaultProblemItem struct {
	Path       string   `json:"path" jsonschema:"path of the file with the problem"`
	Check      string   `json:"check" jsonschema:"the check that found the problem"`
	Detail     string   `json:"detail" jsonschema:"description of the problem in the terms of the file"`
	Address    string   `json:"address,omitempty" jsonschema:"the link target as written, for the ambiguous and dangling checks"`
	Candidates []string `json:"candidates,omitempty" jsonschema:"paths of the notes an ambiguous link could mean"`
}

// VaultProblemsOutput is the output of the vault_problems tool.
type VaultProblemsOutput struct {
	Problems []VaultProblemItem `json:"problems"`
	Checks   []string           `json:"checks" jsonschema:"the checks this answer covers"`
}

func (t *Tools) addVaultList(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "vault_list",
		Title: "List Vaults",
		Description: "List every vault of the installation with its identifier, name and folder path. The vault this " +
			"call works on is marked isCurrent, and a vault whose folder is gone is marked isMissing.",
		Effect:  tools.ReadOnly,
		Servers: []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).listVaults))
}

func (t *Tools) addVaultProblems(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "vault_problems",
		Title: "List Vault Problems",
		Description: "Run checks over the vault and list the problems found: parse for files that could not be " +
			"read, frontmatter for frontmatter that is not valid, ambiguous for links that several notes answer " +
			"to, and dangling for links that reach no note. dangling runs only when named.",
		Effect:  tools.ReadOnly,
		Servers: []tools.Server{tools.Full, tools.Reading},
	}, inVault(t, (*Tools).listVaultProblems))
}

func (t *Tools) listVaults(ctx context.Context, _ VaultListInput) (VaultListOutput, error) {
	vaults, err := t.getVaultEntries(ctx)
	if err != nil {
		return VaultListOutput{}, fmt.Errorf("list vaults: %w", err)
	}
	current := domain.VaultID(tools.GetVault(ctx))
	out := VaultListOutput{Vaults: make([]VaultEntry, 0, len(vaults))}
	for _, v := range vaults {
		out.Vaults = append(out.Vaults, VaultEntry{
			ID:        string(v.ID),
			Name:      v.Name,
			Path:      v.Path,
			IsCurrent: v.ID == current,
			IsMissing: v.IsMissing,
		})
	}
	return out, nil
}

// getVaultEntries returns the vaults of the installation from the vault list, or from the host when the
// tools have no vault list, which reports no vault as missing.
func (t *Tools) getVaultEntries(ctx context.Context) ([]host.VaultEntry, error) {
	if t.list != nil {
		return t.list.GetVaultList(ctx)
	}
	vaults, err := t.host.GetVaults(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]host.VaultEntry, 0, len(vaults))
	for _, v := range vaults {
		out = append(out, host.VaultEntry{Vault: v, IsMissing: false})
	}
	return out, nil
}

func (t *Tools) listVaultProblems(ctx context.Context, in VaultProblemsInput) (VaultProblemsOutput, error) {
	for _, check := range in.Checks {
		if !slices.Contains(getCheckNames(), check) {
			return VaultProblemsOutput{}, fmt.Errorf("unknown check %q: the checks are %s", check, strings.Join(getCheckNames(), ", "))
		}
	}
	problems, err := t.vault.GetProblems(ctx, in.Checks)
	if err != nil {
		return VaultProblemsOutput{}, fmt.Errorf("get vault problems: %w", err)
	}
	out := VaultProblemsOutput{
		Problems: make([]VaultProblemItem, 0, len(problems.Found)),
		Checks:   problems.Checks,
	}
	if out.Checks == nil {
		out.Checks = []string{}
	}
	for _, p := range problems.Found {
		address := ""
		if p.Target.Value != "" {
			address = p.Target.GetWritten()
		}
		out.Problems = append(out.Problems, VaultProblemItem{
			Path:       p.Path,
			Check:      string(p.Kind),
			Detail:     p.Detail,
			Address:    address,
			Candidates: p.Candidates,
		})
	}
	return out, nil
}
