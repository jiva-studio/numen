package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// ResolveLinks finds where addresses written in one note land.
type ResolveLinks struct {
	links port.LinkQueries
}

func NewResolveLinks(links port.LinkQueries) *ResolveLinks {
	return &ResolveLinks{links: links}
}

// Execute answers by the address as it was written, and leaves out one that reaches nothing.
func (u *ResolveLinks) Execute(
	ctx context.Context, v domain.Vault, from string, written []string,
) (map[string]domain.ResolvedLink, error) {
	return u.links.ResolveLinks(ctx, v.ID, from, written)
}
