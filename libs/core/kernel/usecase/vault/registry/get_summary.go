package registry

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// GetSummary finds how much of a vault the index holds.
type GetSummary struct {
	queries noteport.VaultSummaries
}

func NewGetSummary(queries noteport.VaultSummaries) *GetSummary {
	return &GetSummary{queries: queries}
}

// Execute returns the empty summary where no index is bound.
func (u *GetSummary) Execute(ctx context.Context, v domain.Vault) (domain.VaultSummary, error) {
	if u == nil || u.queries == nil {
		return domain.VaultSummary{}, nil
	}
	return u.queries.GetSummary(ctx, v.ID)
}
