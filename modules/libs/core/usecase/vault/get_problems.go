package vault

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/check"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetProblems lists what a vault contains that could not be acted on and was
// not guessed at.
type GetProblems struct {
	checks check.Checks
}

func NewGetProblems(checks check.Checks) *GetProblems {
	return &GetProblems{checks: checks}
}

// NewStandardGetProblems holds the vault to the standard checks.
func NewStandardGetProblems(problems port.ProblemQueries) *GetProblems {
	return NewGetProblems(check.Standard(problems))
}

// ProblemReport is the problems found and the checks that ran to find them.
type ProblemReport struct {
	Ran      []domain.Check
	Problems []domain.VaultProblem
}

// Execute runs the named checks, or every check that is not quiet when none is
// named.
func (u *GetProblems) Execute(ctx context.Context, v domain.Vault, named ...domain.Check) (ProblemReport, error) {
	found, err := u.checks.Run(ctx, v, named...)
	if err != nil {
		return ProblemReport{}, err
	}
	ran := named
	if len(ran) == 0 {
		ran = u.checks.Loud()
	}
	return ProblemReport{Ran: ran, Problems: found}, nil
}
