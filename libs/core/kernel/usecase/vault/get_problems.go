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
	find   VaultFinding
	checks problemChecks
}

func NewGetProblems(checks problemChecks) *GetProblems {
	return &GetProblems{checks: checks}
}

// WithFind sets the lookup that resolves a name, path or identity into a vault.
func (u *GetProblems) WithFind(find VaultFinding) *GetProblems {
	u.find = find
	return u
}

// Problems is what the checks found and the checks that ran.
type Problems struct {
	Ran   []domain.Check
	Found []domain.VaultProblem
}

// ExecuteByName resolves the vault and runs the named checks.
func (u *GetProblems) ExecuteByName(ctx context.Context, nameOrPath string, named ...domain.Check) (Problems, error) {
	v, err := u.find(nameOrPath)
	if err != nil {
		return Problems{}, err
	}
	return u.Execute(ctx, v, named...)
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
