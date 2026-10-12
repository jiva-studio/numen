package vault

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/check"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Overview is everything that says what a vault holds and what in it could not
// be acted on.
type Overview struct {
	Summary  GetSummary
	Problems ListProblems
}

// SummaryQueries is the one question a summary asks of the index.
type SummaryQueries interface {
	// GetSummary is how many notes and headings the vault holds.
	GetSummary(ctx context.Context, vaultID domain.VaultID) (domain.VaultSummary, error)
}

// GetSummary reports how much of a vault has been read.
type GetSummary struct {
	queries SummaryQueries
}

// NewGetSummary is what a vault's size is read out of: the index.
func NewGetSummary(queries SummaryQueries) GetSummary {
	return GetSummary{queries: queries}
}

// Execute returns the summary of the vault.
func (u GetSummary) Execute(ctx context.Context, v domain.Vault) (domain.VaultSummary, error) {
	return u.queries.GetSummary(ctx, v.ID)
}

// NewOverview is what says what a vault holds, over the index. The checks are
// the standard ones where the index answers them, and none where it does not.
func NewOverview(queries port.NoteQueries) Overview {
	var checks check.Checks
	if problems, ok := queries.(port.ProblemQueries); ok && problems != nil {
		checks = check.Standard(problems)
	}
	return Overview{Summary: NewGetSummary(queries), Problems: NewListProblems(checks)}
}

// ProblemsFound is what a look over a vault found, and the checks it covers.
type ProblemsFound struct {
	Problems []domain.VaultProblem
	// Ran is the checks the answer covers, so an empty result states what it
	// covers.
	Ran []domain.Check
}

// ListProblems lists what a vault contains that could not be acted on and was
// not guessed at.
type ListProblems struct {
	checks check.Checks
}

// NewListProblems is what a vault is looked over with: the checks it is held to.
func NewListProblems(checks check.Checks) ListProblems {
	return ListProblems{checks: checks}
}

// Execute runs the named checks, or every check that is not quiet when none is
// named.
func (u ListProblems) Execute(ctx context.Context, v domain.Vault, named ...domain.Check) (ProblemsFound, error) {
	found, err := u.checks.Run(ctx, v, named...)
	if err != nil {
		return ProblemsFound{}, err
	}
	ran := named
	if len(ran) == 0 {
		ran = u.checks.Loud()
	}
	return ProblemsFound{Problems: found, Ran: ran}, nil
}
