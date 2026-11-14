package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// GetLinks lists the links written in one note.
type GetLinks struct {
	links noteport.LinkQueries
}

func NewGetLinks(links noteport.LinkQueries) GetLinks {
	return GetLinks{links: links}
}

// Execute returns the links of the note at the path, each with the note it resolves to.
func (u GetLinks) Execute(ctx context.Context, v domain.Vault, from string) ([]domain.ResolvedLink, error) {
	return u.links.GetLinks(ctx, v.ID, from)
}
