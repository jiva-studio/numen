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

// NewStandardGetProblems holds the vault to the standard checks where the
// queries answer them, and to none where they do not.
func NewStandardGetProblems(queries port.NoteQueries) *GetProblems {
	var checks check.Checks
	if problems, ok := queries.(port.ProblemQueries); ok && problems != nil {
		checks = check.Standard(problems)
	}
	return NewGetProblems(checks)
}

// Execute runs the named checks, or every check that is not quiet when none is
// named.
func (u *GetProblems) Execute(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.VaultProblem, error) {
	return u.checks.Run(ctx, v, named...)
}

// GetDefaultChecks is the checks that run when none is named.
func (u *GetProblems) GetDefaultChecks() []domain.Check {
	return u.checks.Loud()
}
