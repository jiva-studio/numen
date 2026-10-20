package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// CurrentVault is the vault a call is answered about: the vault itself, and where
// it sits on this machine. An agent that can open files joins the folder to a
// path itself; one that cannot reads through a tool.
type CurrentVault struct {
	Vault domain.Vault
	Root  string
}

// getCurrentVault is the vault this call is about. A build that named none answers
// about no vault at all.
func (c Core) getCurrentVault() CurrentVault {
	if c.Showing == nil {
		return CurrentVault{}
	}
	return c.Showing()
}

// ShowOneVault answers with the same vault for as long as the server is served.
func ShowOneVault(v domain.Vault, root string) func() CurrentVault {
	return func() CurrentVault { return CurrentVault{Vault: v, Root: root} }
}

// addVaultTools gives an agent the vault it is working: where it is, and what
// in it could not be read. Both of them read.
func addVaultTools(server *sdk.Server, core Core) {
	addVaultGet(server, core)
	addVaultProblems(server, core)
}

func addVaultGet(server *sdk.Server, core Core) {
	if core.GetSummary == nil {
		return
	}
	addTool(server, &sdk.Tool{
		Name:  "vault_get",
		Title: "Show the vault",
		Description: "Where the vault is and how much of it has been read. The folder " +
			"is the same one named in the instructions; ask again if you have lost it.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, struct {
		Name     string `json:"name"`
		Folder   string `json:"folder" jsonschema:"where the vault is on this machine; join a note's path to it to open the file"`
		ID       string `json:"id"`
		Notes    int    `json:"notes"`
		Headings int    `json:"headings"`
	}, error) {
		type out = struct {
			Name     string `json:"name"`
			Folder   string `json:"folder" jsonschema:"where the vault is on this machine; join a note's path to it to open the file"`
			ID       string `json:"id"`
			Notes    int    `json:"notes"`
			Headings int    `json:"headings"`
		}
		shown := core.getCurrentVault()
		summary, err := core.GetSummary.Execute(ctx, shown.Vault)
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{
			Name:     shown.Vault.Name,
			Folder:   shown.Root,
			ID:       string(shown.Vault.ID),
			Notes:    summary.Notes,
			Headings: summary.Headings,
		}, nil
	})
}

func addVaultProblems(server *sdk.Server, core Core) {
	if core.GetProblems == nil {
		return
	}
	addTool(server, &sdk.Tool{
		Name:  "vault_problems",
		Title: "List scan problems",
		Description: "What the vault contains that could not be acted on and was not " +
			"guessed at. Each problem names the note somebody would open to settle it, " +
			"and the check that noticed it: `parse` for what one file got wrong, " +
			"`frontmatter` for a block that is not YAML, `ambiguous` for a link that " +
			"reaches several notes. `dangling` — a link that reaches nothing — is left " +
			"out unless you ask for it by name, because a vault being written is full " +
			"of them. This is the work list for tidying a vault up.",
	}, newHandler(func(ctx context.Context, in vaultProblemsInput) (vaultProblemsOutput, error) {
		return listVaultProblems(ctx, core, in)
	}))
}

type vaultProblemsInput struct {
	Checks []string `json:"checks,omitempty" jsonschema:"run only these checks: parse, frontmatter, ambiguous, dangling"`
}

type vaultProblemsOutput struct {
	Problems []Problem `json:"problems"`
	Ran      []string  `json:"ran" jsonschema:"the checks this answer covers"`
}

func listVaultProblems(ctx context.Context, core Core, in vaultProblemsInput) (vaultProblemsOutput, error) {
	named := make([]domain.Check, 0, len(in.Checks))
	for _, name := range in.Checks {
		named = append(named, domain.Check(name))
	}
	report, err := core.GetProblems.Execute(ctx, core.getCurrentVault().Vault, named...)
	if err != nil {
		return vaultProblemsOutput{}, err
	}

	res := vaultProblemsOutput{Problems: make([]Problem, 0, len(report.Found)), Ran: make([]string, 0, len(report.Ran))}
	for _, name := range report.Ran {
		res.Ran = append(res.Ran, string(name))
	}
	for _, p := range report.Found {
		res.Problems = append(res.Problems, Problem{
			Path:       p.Path,
			Check:      string(p.Kind),
			Detail:     p.Detail,
			Address:    p.Target.String(),
			Candidates: p.Candidates,
		})
	}
	return res, nil
}

// Problem is one thing worth a person's attention, and which check noticed it.
type Problem struct {
	Path   string `json:"path" jsonschema:"the note somebody would open to settle this"`
	Check  string `json:"check"`
	Detail string `json:"detail"`
	// Address is what the link says, for the checks that are about one.
	Address string `json:"address,omitempty"`
	// Candidates is the notes that answer to it, when several do.
	Candidates []string `json:"candidates,omitempty"`
}
