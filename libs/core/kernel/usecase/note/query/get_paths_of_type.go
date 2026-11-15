package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

type typedPaths interface {
	GetPathsOfType(ctx context.Context, vaultID domain.VaultID, noteType domain.NoteType) ([]string, error)
}

// GetPathsOfType lists the notes of one type.
type GetPathsOfType struct {
	queries typedPaths
}

func NewGetPathsOfType(queries typedPaths) GetPathsOfType {
	return GetPathsOfType{queries: queries}
}

// Execute returns the path of every note of the type the vault holds.
func (u GetPathsOfType) Execute(ctx context.Context, v domain.Vault, noteType domain.NoteType) ([]string, error) {
	return u.queries.GetPathsOfType(ctx, v.ID, noteType)
}
