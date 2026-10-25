package note

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

// Execute answers by the address as it was written, and leaves out one that reaches nothing.
func (u *ResolveLinks) Execute(
	ctx context.Context, v domain.Vault, from string, targets []string,
) (map[string]domain.ResolvedLink, error) {
	return u.links.ResolveLinks(ctx, v.ID, from, targets)
}
