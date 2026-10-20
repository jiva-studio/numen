package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetTypes finds what each of the notes at some paths is.
type GetTypes struct {
	queries port.NoteQueries
}

func NewGetTypes(queries port.NoteQueries) *GetTypes {
	return &GetTypes{queries: queries}
}

// Execute answers by path, and leaves out a path that names no note.
func (u *GetTypes) Execute(ctx context.Context, v domain.Vault, paths []string) (map[string]domain.NoteType, error) {
	return u.queries.GetTypes(ctx, v.ID, paths)
}
