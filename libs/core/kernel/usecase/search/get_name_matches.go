package search

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// GetNameMatches finds the names in a vault that match the words typed: a note's own title and the headings inside notes.
type GetNameMatches struct {
	queries noteport.NameMatches
}

func NewGetNameMatches(queries noteport.NameMatches) *GetNameMatches {
	return &GetNameMatches{queries: queries}
}

func (u *GetNameMatches) Execute(
	ctx context.Context, v domain.Vault, query string, limit int,
) ([]domain.NameMatch, error) {
	return u.queries.GetNameMatches(ctx, v.ID, query, limit)
}
