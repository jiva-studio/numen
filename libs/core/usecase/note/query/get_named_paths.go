package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
)

// GetNamedPaths returns the notes that a link's name resolves to.
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
