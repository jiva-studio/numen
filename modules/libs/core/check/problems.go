package check

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// parseCheck is what reading one file turned up, and the scan already knows it:
// a link with no role, a role nobody decided on, an identifier that is not one.
//
// Nothing is worked out here. The rule that decided these ran when the note was
// parsed, which is the only place that can see what the file says without
// reading it again.
type parseCheck struct{ queries port.ProblemQueries }

func (parseCheck) Name() domain.Check { return domain.CheckParse }
func (parseCheck) Quiet() bool        { return false }

func (c parseCheck) Look(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error) {
	noted, err := c.queries.Noted(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.VaultProblem, 0, len(noted))
	for _, n := range noted {
		out = append(out, domain.VaultProblem{
			Path: n.Path, Kind: domain.CheckParse, Detail: n.Detail,
		})
	}
	return out, nil
}

// frontmatterCheck is a block between the delimiters that is not YAML.
//
// The note is indexed anyway — its text is readable either way — but nothing
// may write to it: changing a key in a block the application could not read
// means guessing at the rest of it.
type frontmatterCheck struct{ queries port.ProblemQueries }

func (frontmatterCheck) Name() domain.Check { return domain.CheckFrontmatter }
func (frontmatterCheck) Quiet() bool        { return false }

func (c frontmatterCheck) Look(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error) {
	unreadable, err := c.queries.Unreadable(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.VaultProblem, 0, len(unreadable))
	for _, n := range unreadable {
		out = append(out, domain.VaultProblem{
			Path: n.Path,
			Kind: domain.CheckFrontmatter,
			Detail: "the frontmatter is not YAML, so nothing may be written here: " +
				n.Detail,
		})
	}
	return out, nil
}
