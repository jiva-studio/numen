package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// addVaultTools gives an agent the vault it is working: where it is, what it
// holds under one name, and what in it could not be read. Every one of them
// reads.
func addVaultTools(server *sdk.Server, core Core) {
	addVaultGet(server, core)
	addVaultProblems(server, core)
	addVaultNamed(server, core)
}

func addVaultGet(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
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
		shown := core.shown()
		summary, err := core.Notes.Queries.Summary(ctx, string(shown.Vault.ID))
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
	sdk.AddTool(server, &sdk.Tool{
		Name:  "vault_problems",
		Title: "List what a scan could not act on",
		Description: "What the vault contains that could not be acted on and was not " +
			"guessed at. Each problem names the note somebody would open to settle it, " +
			"and the check that noticed it: `parse` for what one file got wrong, " +
			"`frontmatter` for a block that is not YAML, `ambiguous` for a link that " +
			"reaches several notes. `dangling` — a link that reaches nothing — is left " +
			"out unless you ask for it by name, because a vault being written is full " +
			"of them. This is the work list for tidying a vault up.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Checks []string `json:"checks,omitempty" jsonschema:"run only these checks: parse, frontmatter, ambiguous, dangling"`
	}) (*sdk.CallToolResult, struct {
		Problems []Problem `json:"problems"`
		Ran      []string  `json:"ran" jsonschema:"the checks this answer covers"`
	}, error) {
		type out = struct {
			Problems []Problem `json:"problems"`
			Ran      []string  `json:"ran" jsonschema:"the checks this answer covers"`
		}
		named := make([]domain.Check, 0, len(in.Checks))
		for _, name := range in.Checks {
			named = append(named, domain.Check(name))
		}
		found, err := core.Notes.Problems.Run(ctx, core.shown().Vault, named...)
		if err != nil {
			return nil, out{}, err
		}

		res := out{Problems: make([]Problem, 0, len(found)), Ran: in.Checks}
		if len(res.Ran) == 0 {
			// Saying which checks an empty answer covers is the difference
			// between "nothing is wrong" and "nothing I looked at is wrong".
			for _, name := range core.Notes.Problems.Loud() {
				res.Ran = append(res.Ran, string(name))
			}
		}
		for _, p := range found {
			res.Problems = append(res.Problems, Problem{
				Path:       p.Path,
				Check:      string(p.Kind),
				Detail:     p.Detail,
				Address:    p.Target.String(),
				Candidates: p.Candidates,
			})
		}
		return nil, res, nil
	})
}

func addVaultNamed(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "vault_named",
		Title: "Find notes by name",
		Description: "Every note filed under one name. More than one means a link " +
			"written by that name is ambiguous and reaches the nearest of them, which " +
			"can change when either note is moved.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Name string `json:"name" jsonschema:"a note's filename without its extension"`
	}) (*sdk.CallToolResult, struct {
		Paths []string `json:"paths"`
	}, error) {
		type out = struct {
			Paths []string `json:"paths"`
		}
		paths, err := core.Notes.Queries.Named(ctx, string(core.shown().Vault.ID), domain.LinkName(in.Name))
		if err != nil {
			return nil, out{}, err
		}
		return nil, out{Paths: paths}, nil
	})
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
