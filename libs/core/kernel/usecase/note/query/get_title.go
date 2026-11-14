package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// GetTitle finds what the note at a path is called.
type GetTitle struct {
	queries noteport.NoteRefs
}

func NewGetTitle(queries noteport.NoteRefs) *GetTitle {
	if queries == nil {
		queries = noNotes{}
	}
	return &GetTitle{queries: queries}
}

// Execute returns "" for an empty path, an unknown note or an index error.
func (u *GetTitle) Execute(ctx context.Context, v domain.Vault, path string) string {
	if u == nil || path == "" {
		return ""
	}
	found, err := u.queries.GetNotes(ctx, v.ID, []string{path})
	if err != nil {
		return ""
	}
	return found[path].Title
}

// noNotes is the index of a build that has none: it holds no note.
type noNotes struct{}

func (noNotes) GetNotes(context.Context, domain.VaultID, []string) (map[string]domain.NoteRef, error) {
	return nil, nil
}
