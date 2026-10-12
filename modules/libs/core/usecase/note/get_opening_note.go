package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetOpeningNote finds the note to show when nothing else has been chosen.
type GetOpeningNote struct {
	queries port.NoteQueries
}

func NewGetOpeningNote(queries port.NoteQueries) GetOpeningNote {
	return GetOpeningNote{queries: queries}
}

// Execute returns nil for a vault holding no note.
func (u GetOpeningNote) Execute(ctx context.Context, v domain.Vault) (*domain.NoteRef, error) {
	ref, found, err := u.queries.GetOpeningNote(ctx, v.ID)
	if err != nil || !found {
		return nil, err
	}
	return &ref, nil
}
