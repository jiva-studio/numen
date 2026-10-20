package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetTitle finds what the note at a path is called.
type GetTitle struct {
	queries port.NoteQueries
}

func NewGetTitle(queries port.NoteQueries) *GetTitle {
	return &GetTitle{queries: queries}
}

// Execute returns "" for an empty path, an unknown note or an index error.
func (u *GetTitle) Execute(ctx context.Context, v domain.Vault, path string) string {
	if u == nil || u.queries == nil || path == "" {
		return ""
	}
	found, err := u.queries.GetNotes(ctx, v.ID, []string{path})
	if err != nil {
		return ""
	}
	return found[path].Title
}
