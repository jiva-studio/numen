package query

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/noteport"
)

// GetTitle finds what the note at a path is called.
type GetTitle struct {
	queries noteport.NoteQueries
}

func NewGetTitle(queries noteport.NoteQueries) *GetTitle {
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
