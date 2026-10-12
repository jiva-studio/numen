package vault

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/check"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ListProblems lists what a vault contains that could not be acted on and was
// not guessed at.
type ListProblems struct {
	checks check.Checks
}

func NewListProblems(checks check.Checks) *ListProblems {
	return &ListProblems{checks: checks}
}

// NewStandardListProblems holds the vault to the standard checks where the
// queries answer them, and to none where they do not.
func NewStandardListProblems(queries port.NoteQueries) *ListProblems {
	var checks check.Checks
	if problems, ok := queries.(port.ProblemQueries); ok && problems != nil {
		checks = check.Standard(problems)
	}
	return NewListProblems(checks)
}

// Execute runs the named checks, or every check that is not quiet when none is
// named.
func (u *ListProblems) Execute(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.VaultProblem, error) {
	return u.checks.Run(ctx, v, named...)
}

// GetDefaultChecks is the checks that run when none is named.
func (u *ListProblems) GetDefaultChecks() []domain.Check {
	return u.checks.Loud()
}
