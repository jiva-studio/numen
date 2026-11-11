package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// problemChecks runs the checks over a vault.
type problemChecks interface {
	Run(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.VaultProblem, error)
	GetDefaultNames() []domain.Check
}

// GetProblems lists what a vault contains that could not be acted on and was
// not guessed at.
type GetProblems struct {
	checks problemChecks
}

func NewGetProblems(checks problemChecks) *GetProblems {
	return &GetProblems{checks: checks}
}

// Problems is what the checks found and the checks that ran.
type Problems struct {
	Ran   []domain.Check
	Found []domain.VaultProblem
}

// GetProblemsByName lists the problems of the vault a person names.
type GetProblemsByName struct {
	find     VaultFinding
	problems *GetProblems
}

// NewGetProblemsByName creates the use case that resolves the vault with find and lists its problems with problems.
func NewGetProblemsByName(find VaultFinding, problems *GetProblems) GetProblemsByName {
	return GetProblemsByName{find: find, problems: problems}
}

// ExecuteByName resolves the vault and runs the named checks.
func (u GetProblemsByName) ExecuteByName(ctx context.Context, nameOrPath string, named ...domain.Check) (Problems, error) {
	v, err := u.find(nameOrPath)
	if err != nil {
		return Problems{}, err
	}
	return u.problems.Execute(ctx, v, named...)
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
		ran = u.checks.GetDefaultNames()
	}
	return Problems{Ran: ran, Found: found}, nil
}
