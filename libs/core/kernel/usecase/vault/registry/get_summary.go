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
	if queries == nil {
		queries = noSummaries{}
	}
	return &GetSummary{queries: queries}
}

// Execute returns the empty summary where no index is bound.
func (u *GetSummary) Execute(ctx context.Context, v domain.Vault) (domain.VaultSummary, error) {
	if u == nil {
		return domain.VaultSummary{}, nil
	}
	return u.queries.GetSummary(ctx, v.ID)
}

// noSummaries is the index of a build that has none: it holds nothing of any vault.
type noSummaries struct{}

func (noSummaries) GetSummary(context.Context, domain.VaultID) (domain.VaultSummary, error) {
	return domain.VaultSummary{}, nil
}
