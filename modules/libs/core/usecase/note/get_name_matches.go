package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetNameMatches finds the notes and headings whose names match what was typed.
type GetNameMatches struct {
	queries port.NoteQueries
}

func NewGetNameMatches(queries port.NoteQueries) GetNameMatches {
	return GetNameMatches{queries: queries}
}

func (u GetNameMatches) Execute(ctx context.Context, v domain.Vault, query string, limit int) ([]domain.NameMatch, error) {
	return u.queries.GetNameMatches(ctx, v.ID, query, limit)
}
