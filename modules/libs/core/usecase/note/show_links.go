package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// NoteLinks is what one note is joined to: what it points at, and what points
// at it. Both halves together, because looking at one without the other is how
// a graph gets misread — a note with no outgoing links may still be the centre
// of the vault.
type NoteLinks struct {
	Links     []domain.ResolvedLink
	Backlinks []domain.ResolvedLink
}

// ShowLinks gathers them for one note.
type ShowLinks struct {
	Links port.LinkQueries
}

// NewShowLinks is what both halves are asked of: the index that answers what a
// note points at and what points at it.
func NewShowLinks(links port.LinkQueries) ShowLinks {
	return ShowLinks{Links: links}
}

func (u ShowLinks) Execute(ctx context.Context, v domain.Vault, path string) (NoteLinks, error) {
	links, err := u.Links.Links(ctx, v.ID, path)
	if err != nil {
		return NoteLinks{}, err
	}
	backlinks, err := u.Links.Backlinks(ctx, v.ID, path)
	if err != nil {
		return NoteLinks{}, err
	}
	return NoteLinks{Links: links, Backlinks: backlinks}, nil
}
