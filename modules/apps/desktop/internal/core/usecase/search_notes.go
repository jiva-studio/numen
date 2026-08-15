package usecase

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// SearchNotes answers a query within one vault.
//
// Scoping to a vault is not a parameter the caller may forget: one database
// holds every vault, and a search that crossed them would mix a work vault into
// a personal one without failing.
type SearchNotes struct {
	Notes port.NoteRepository
	Limit int
}

const defaultLimit = 20

func (u SearchNotes) Execute(ctx context.Context, v domain.Vault, query string) ([]domain.Hit, error) {
	limit := u.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	return u.Notes.Search(ctx, v.ID, query, limit)
}
