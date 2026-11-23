package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// ListBacklinks lists the links written in other notes that resolve to one note.
type ListBacklinks struct {
	links noteport.LinkQueries
}

// NewListBacklinks builds the use case over the link queries of the index.
func NewListBacklinks(links noteport.LinkQueries) ListBacklinks {
	return ListBacklinks{links: links}
}

// Execute returns every link that points at the note. Each carries the role as
// written, which is relative to the note it is written in, and the role of that
// note relative to the one asked about.
func (u ListBacklinks) Execute(ctx context.Context, v domain.Vault, path string) ([]domain.Backlink, error) {
	links, err := u.links.ListBacklinks(ctx, v.ID, path)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Backlink, 0, len(links))
	for _, l := range links {
		inverse, _ := domain.GetInverseRole(l.Role)
		out = append(out, domain.Backlink{ResolvedLink: l, SourceRole: inverse})
	}
	return out, nil
}
