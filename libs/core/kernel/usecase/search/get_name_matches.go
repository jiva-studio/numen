package search

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// GetNameMatches finds the names in a vault that match the words typed: a note's own title and the headings inside notes.
type GetNameMatches struct {
	queries NameMatches
}

func NewGetNameMatches(queries NameMatches) *GetNameMatches {
	return &GetNameMatches{queries: queries}
}

func (u *GetNameMatches) Execute(
	ctx context.Context, v domain.Vault, query string, limit int,
) ([]domain.NameMatch, error) {
	return u.queries.GetNameMatches(ctx, v.ID, query, limit)
}
