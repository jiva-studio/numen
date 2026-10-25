package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
)

// GetTypes finds what each of the notes at some paths is.
type GetTypes struct {
	queries noteport.NoteQueries
}

func NewGetTypes(queries noteport.NoteQueries) *GetTypes {
	return &GetTypes{queries: queries}
}

// Execute answers by path, and leaves out a path that names no note.
func (u *GetTypes) Execute(ctx context.Context, v domain.Vault, paths []string) (map[string]domain.NoteType, error) {
	return u.queries.GetTypes(ctx, v.ID, paths)
}
