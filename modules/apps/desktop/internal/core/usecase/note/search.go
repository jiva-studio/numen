package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Search answers a query within one vault.
//
// Scoping to a vault is not a parameter the caller may forget: one database
// holds every vault, and a search that crossed them would mix a work vault into
// a personal one without failing.
type Search struct {
	Notes port.NoteQueries
	Limit int
}

const defaultLimit = 20

func (u Search) Execute(ctx context.Context, v domain.Vault, query string) ([]domain.NoteMatch, error) {
	limit := u.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	return u.Notes.Search(ctx, v.ID, query, limit)
}
