package vault

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetSummary reports how much of a vault has been read.
type GetSummary struct {
	queries port.NoteQueries
}

func NewGetSummary(queries port.NoteQueries) *GetSummary {
	return &GetSummary{queries: queries}
}

func (u *GetSummary) Execute(ctx context.Context, v domain.Vault) (domain.VaultSummary, error) {
	return u.queries.GetSummary(ctx, v.ID)
}
