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
