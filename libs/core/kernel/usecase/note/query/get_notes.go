package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// GetNotes reads what the index holds of the notes at some paths.
type GetNotes struct {
	queries noteport.NoteRefs
}

func NewGetNotes(queries noteport.NoteRefs) GetNotes {
	return GetNotes{queries: queries}
}

// Execute answers by path, and leaves out a path that names no note.
func (u GetNotes) Execute(ctx context.Context, v domain.Vault, paths []string) (map[string]domain.NoteRef, error) {
	return u.queries.GetNotes(ctx, v.ID, paths)
}
