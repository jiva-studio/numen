package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type typedPaths interface {
	ListPathsOfType(ctx context.Context, vaultID domain.VaultID, noteType domain.NoteType) ([]string, error)
}

type ListPathsOfType struct {
	queries typedPaths
}

func NewListPathsOfType(queries typedPaths) ListPathsOfType {
	return ListPathsOfType{queries: queries}
}

// Execute returns the path of every note of the type the vault holds.
func (u ListPathsOfType) Execute(ctx context.Context, v domain.Vault, noteType domain.NoteType) ([]string, error) {
	return u.queries.ListPathsOfType(ctx, v.ID, noteType)
}
