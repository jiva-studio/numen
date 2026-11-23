package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type parseProblems interface {
	ListParseProblems(ctx context.Context, vaultID domain.VaultID) ([]domain.VaultProblem, error)
}

// ListParseProblems lists what the scan found while parsing the notes of a vault.
type ListParseProblems struct {
	queries parseProblems
}

func NewListParseProblems(queries parseProblems) *ListParseProblems {
	return &ListParseProblems{queries: queries}
}

// Execute returns the problems the index recorded for the vault when it parsed each file.
func (u *ListParseProblems) Execute(ctx context.Context, v domain.Vault) ([]domain.VaultProblem, error) {
	return u.queries.ListParseProblems(ctx, v.ID)
}
