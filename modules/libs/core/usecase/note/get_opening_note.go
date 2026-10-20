package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetOpeningNote finds the note to show when nothing else has been chosen.
type GetOpeningNote struct {
	queries port.NoteQueries
}

func NewGetOpeningNote(queries port.NoteQueries) *GetOpeningNote {
	return &GetOpeningNote{queries: queries}
}

// Execute answers false when the vault holds no note to open.
func (u *GetOpeningNote) Execute(ctx context.Context, v domain.Vault) (domain.NoteRef, bool, error) {
	return u.queries.GetOpeningNote(ctx, v.ID)
}
