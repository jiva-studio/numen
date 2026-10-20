package vault

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetSummary finds how much of a vault the index holds.
type GetSummary struct {
	queries port.NoteQueries
}

func NewGetSummary(queries port.NoteQueries) *GetSummary {
	return &GetSummary{queries: queries}
}

// Execute returns the empty summary where no index is bound.
func (u *GetSummary) Execute(ctx context.Context, v domain.Vault) (domain.VaultSummary, error) {
	if u == nil || u.queries == nil {
		return domain.VaultSummary{}, nil
	}
	return u.queries.GetSummary(ctx, v.ID)
}
