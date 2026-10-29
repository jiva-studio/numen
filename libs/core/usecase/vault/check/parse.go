package check

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/problemport"
)

// parseCheck is what reading one file found, which the scan already has: a link
// with no role, an unassigned role, an identifier that is invalid.
//
// Nothing is computed here. The rule that decided these ran when the note was
// parsed, which is the only place that can see what the file contains without
// reading it again.
type parseCheck struct {
	queries problemport.ProblemQueries
}

func (parseCheck) Name() domain.Check { return domain.CheckParse }
func (parseCheck) IsOptIn() bool      { return false }

func (c parseCheck) FindProblems(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error) {
	noted, err := c.queries.GetParseProblems(ctx, v.ID)
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
