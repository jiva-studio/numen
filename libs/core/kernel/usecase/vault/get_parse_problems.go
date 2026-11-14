package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

type parseProblems interface {
	GetParseProblems(ctx context.Context, vaultID domain.VaultID) ([]domain.VaultProblem, error)
}

// GetParseProblems lists what the scan found while parsing the notes of a vault.
type GetParseProblems struct {
	queries parseProblems
}

func NewGetParseProblems(queries parseProblems) *GetParseProblems {
	return &GetParseProblems{queries: queries}
}

// Execute returns the problems the index recorded for the vault when it parsed each file.
func (u *GetParseProblems) Execute(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error) {
	return u.queries.GetParseProblems(ctx, v.ID)
}
