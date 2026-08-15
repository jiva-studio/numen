package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// ListProblems is the view a vault needs because the application refuses to
// guess: a link with no role, a role nobody decided on, frontmatter that will
// not parse. None of it stops a scan, so without somewhere to look it would all
// be invisible.
type ListProblems struct {
	Problems port.ProblemQueries
}

func (u ListProblems) Execute(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error) {
	return u.Problems.Problems(ctx, v.ID)
}
