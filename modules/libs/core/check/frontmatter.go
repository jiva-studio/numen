package check

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

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
