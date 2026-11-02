package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
)

// ResolveLinks finds where addresses written in one note land.
type ResolveLinks struct {
	links noteport.LinkQueries
}

func NewResolveLinks(links noteport.LinkQueries) *ResolveLinks {
	return &ResolveLinks{links: links}
}

// Execute resolves each address as written, and omits an address that resolves to nothing.
func (u *ResolveLinks) Execute(
	ctx context.Context, v domain.Vault, from string, targets []string,
) (map[string]domain.ResolvedLink, error) {
	return u.links.ResolveLinks(ctx, v.ID, from, targets)
}
