package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/usecase/vault/check"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/problemport"
)

// GetProblems lists what a vault contains that could not be acted on and was
// not guessed at.
type GetProblems struct {
	checks check.Checks
}

func NewGetProblems(checks check.Checks) *GetProblems {
	return &GetProblems{checks: checks}
}

// NewStandardGetProblems holds the vault to the standard checks, and to none where there are no problem queries.
func NewStandardGetProblems(problems problemport.ProblemQueries) *GetProblems {
	if problems == nil {
		return NewGetProblems(check.Checks{})
	}
	return NewGetProblems(check.Standard(problems))
}

// Problems is what the checks found and the checks that ran.
type Problems struct {
	Ran   []domain.Check
	Found []domain.VaultProblem
}

// Execute runs the named checks, or every check that is not quiet when none is
// named.
func (u *GetProblems) Execute(ctx context.Context, v domain.Vault, named ...domain.Check) (Problems, error) {
	found, err := u.checks.Run(ctx, v, named...)
	if err != nil {
		return Problems{}, err
	}
	ran := named
	if len(ran) == 0 {
		ran = u.checks.Loud()
	}
	return Problems{Ran: ran, Found: found}, nil
}
