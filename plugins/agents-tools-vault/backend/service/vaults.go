package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// VaultEntry is a vault of the installation.
type VaultEntry struct {
	ID        string `json:"id" jsonschema:"vault identifier, which stays the same when its folder moves"`
	Name      string `json:"name" jsonschema:"vault display name"`
	Path      string `json:"path" jsonschema:"vault root directory on this machine"`
	IsCurrent bool   `json:"isCurrent" jsonschema:"true for the current vault"`
	IsMissing bool   `json:"isMissing,omitempty" jsonschema:"true when the vault folder does not exist"`
}

// VaultListInput is the input of the vault_list tool.
type VaultListInput struct{}

// VaultListOutput is the output of the vault_list tool.
type VaultListOutput struct {
	Vaults []VaultEntry `json:"vaults"`
}

// VaultProblemListInput is the input of the vault_problem_list tool.
type VaultProblemListInput struct {
	Checks []string `json:"checks,omitempty" jsonschema:"the checks to run: parse, frontmatter, ambiguous and dangling; parse, frontmatter and ambiguous run when left out"`
}

// VaultProblemItem is a problem found in one file of the vault.
type VaultProblemItem struct {
	Path       string   `json:"path" jsonschema:"path of the file with the problem"`
	Check      string   `json:"check" jsonschema:"the check that found the problem"`
	Detail     string   `json:"detail" jsonschema:"description of the problem"`
	Address    string   `json:"address,omitempty" jsonschema:"the link target as written, for the ambiguous and dangling checks"`
	Candidates []string `json:"candidates,omitempty" jsonschema:"paths of the notes an ambiguous link could resolve to"`
}

// VaultProblemListOutput is the output of the vault_problem_list tool.
type VaultProblemListOutput struct {
	Problems []VaultProblemItem `json:"problems"`
	Checks   []string           `json:"checks" jsonschema:"the checks this result covers"`
}

func (t *Tools) addVaultList(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "vault_list",
		Title: "List Vaults",
		Description: "List every vault of the installation with its identifier, name and folder path. The current " +
			"vault is marked isCurrent, and a vault whose folder is missing is marked isMissing.",
		Effect:       tools.ReadOnly,
		Servers:      []tools.Server{tools.Full, tools.Reading},
		Instructions: []string{vaultListSentence},
	}, inVault(t, (*Tools).listVaults))
}

func (t *Tools) addVaultProblemList(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "vault_problem_list",
		Title: "List Vault Problems",
		Description: "Run checks over the vault and list the problems found: parse for files that could not be " +
			"read, frontmatter for frontmatter that is not valid, ambiguous for links that resolve to several " +
			"notes, and dangling for links that resolve to no note. dangling runs only when named.",
		Effect:       tools.ReadOnly,
		Servers:      []tools.Server{tools.Full, tools.Reading},
		Instructions: []string{vaultProblemListSentence},
	}, inVault(t, (*Tools).listVaultProblems))
}

func (t *Tools) listVaults(ctx context.Context, _ VaultListInput) (VaultListOutput, error) {
	vaults, err := t.getVaultEntries(ctx)
	if err != nil {
		return VaultListOutput{}, err
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
// tools have no vault list, where IsMissing is always false.
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

func (t *Tools) listVaultProblems(ctx context.Context, in VaultProblemListInput) (VaultProblemListOutput, error) {
	problems, err := t.vault.GetProblems(ctx, in.Checks)
	if err != nil {
		return VaultProblemListOutput{}, err
	}
	out := VaultProblemListOutput{
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
