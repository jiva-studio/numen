package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// ListLinks lists the links written in one note.
type ListLinks struct {
	links noteport.LinkQueries
}

func NewListLinks(links noteport.LinkQueries) ListLinks {
	return ListLinks{links: links}
}

// Execute returns the links of the note at the path, each with the note it resolves to.
func (u ListLinks) Execute(ctx context.Context, v domain.Vault, from string) ([]domain.ResolvedLink, error) {
	return u.links.ListLinks(ctx, v.ID, from)
}
