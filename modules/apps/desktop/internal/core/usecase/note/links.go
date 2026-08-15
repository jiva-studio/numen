package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Connections is what one note is joined to: what it points at, and what points
// at it. Both halves together, because looking at one without the other is how
// a graph gets misread — a note with no outgoing links may still be the centre
// of the vault.
type Connections struct {
	Links     []domain.ResolvedLink
	Backlinks []domain.ResolvedLink
}

// ShowConnections gathers them for one note.
type ShowConnections struct {
	Links port.LinkQueries
}

func (u ShowConnections) Execute(ctx context.Context, v domain.Vault, path string) (Connections, error) {
	links, err := u.Links.Links(ctx, v.ID, path)
	if err != nil {
		return Connections{}, err
	}
	backlinks, err := u.Links.Backlinks(ctx, v.ID, path)
	if err != nil {
		return Connections{}, err
	}
	return Connections{Links: links, Backlinks: backlinks}, nil
}
