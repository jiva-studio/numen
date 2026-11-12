package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// GetOpeningNote finds the note to show when nothing else has been chosen.
type GetOpeningNote struct {
	queries noteport.OpeningNote
}

func NewGetOpeningNote(queries noteport.OpeningNote) *GetOpeningNote {
	return &GetOpeningNote{queries: queries}
}

// Execute answers false when the vault holds no note to open.
func (u *GetOpeningNote) Execute(ctx context.Context, v domain.Vault) (domain.NoteRef, bool, error) {
	return u.queries.GetOpeningNote(ctx, v.ID)
}
