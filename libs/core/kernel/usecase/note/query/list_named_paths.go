package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ListNamedPaths returns the notes that a link's name resolves to.
type ListNamedPaths struct {
	queries NamedPaths
}

func NewListNamedPaths(queries NamedPaths) *ListNamedPaths {
	return &ListNamedPaths{queries: queries}
}

// Execute returns the paths of every note filed under the name, and none for an
// empty name or an unbound index.
func (u *ListNamedPaths) Execute(ctx context.Context, v domain.Vault, name string) ([]string, error) {
	if u == nil || name == "" {
		return nil, nil
	}
	return u.queries.ListNamedPaths(ctx, v.ID, name)
}
