package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/noteport"
)

// GetOpeningNote finds the note to show when nothing else has been chosen.
type GetOpeningNote struct {
	queries noteport.NoteQueries
}

func NewGetOpeningNote(queries noteport.NoteQueries) *GetOpeningNote {
	return &GetOpeningNote{queries: queries}
}

// Execute answers false when the vault holds no note to open.
func (u *GetOpeningNote) Execute(ctx context.Context, v domain.Vault) (domain.NoteRef, bool, error) {
	return u.queries.GetOpeningNote(ctx, v.ID)
}
