package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/noteport"
)

// GetNamedPaths finds the notes a link's name reaches.
type GetNamedPaths struct {
	queries noteport.NoteQueries
}

func NewGetNamedPaths(queries noteport.NoteQueries) *GetNamedPaths {
	return &GetNamedPaths{queries: queries}
}

// Execute returns the paths of every note filed under the name, and none for an
// empty name or an unbound index.
func (u *GetNamedPaths) Execute(ctx context.Context, v domain.Vault, name string) ([]string, error) {
	if u == nil || u.queries == nil || name == "" {
		return nil, nil
	}
	return u.queries.GetNamedPaths(ctx, v.ID, domain.LinkName(name))
}
