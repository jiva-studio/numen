package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// GetNamedPaths returns the notes that a link's name resolves to.
type GetNamedPaths struct {
	queries noteport.NamedPaths
}

func NewGetNamedPaths(queries noteport.NamedPaths) *GetNamedPaths {
	if queries == nil {
		queries = noNames{}
	}
	return &GetNamedPaths{queries: queries}
}

// Execute returns the paths of every note filed under the name, and none for an
// empty name or an unbound index.
func (u *GetNamedPaths) Execute(ctx context.Context, v domain.Vault, name string) ([]string, error) {
	if u == nil || name == "" {
		return nil, nil
	}
	return u.queries.GetNamedPaths(ctx, v.ID, domain.LinkName(name))
}

// noNames is the index of a build that has none: no name resolves.
type noNames struct{}

func (noNames) GetNamedPaths(context.Context, domain.VaultID, string) ([]string, error) {
	return nil, nil
}
