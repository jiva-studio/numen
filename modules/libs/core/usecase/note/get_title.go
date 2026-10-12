package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetTitle finds what the note at a path is called.
type GetTitle struct {
	queries port.NoteQueries
}

func NewGetTitle(queries port.NoteQueries) GetTitle {
	return GetTitle{queries: queries}
}

// Execute returns no title for an empty path, a path naming no note and an
// index that cannot answer.
func (u GetTitle) Execute(ctx context.Context, v domain.Vault, path string) string {
	if path == "" {
		return ""
	}
	found, err := u.queries.GetNotes(ctx, v.ID, []string{path})
	if err != nil {
		return ""
	}
	return found[path].Title
}
