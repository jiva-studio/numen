package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
)

// GetBacklinks lists the links written in other notes that resolve to one note.
type GetBacklinks struct {
	links noteport.LinkQueries
}

// NewGetBacklinks builds the use case over the link queries of the index.
func NewGetBacklinks(links noteport.LinkQueries) GetBacklinks {
	return GetBacklinks{links: links}
}

// Execute returns every link that points at the note. Each carries the role as
// written, which is relative to the note it is written in, and the role of that
// note relative to the one asked about.
func (u GetBacklinks) Execute(ctx context.Context, v domain.Vault, path string) ([]domain.Backlink, error) {
	links, err := u.links.GetBacklinks(ctx, v.ID, path)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Backlink, 0, len(links))
	for _, l := range links {
		inverse, _ := domain.GetInverseRole(l.Role)
		out = append(out, domain.Backlink{ResolvedLink: l, RoleFromHere: inverse})
	}
	return out, nil
}
