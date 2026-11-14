package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// IsVaultIndexed reports whether the index already contains a vault.
type IsVaultIndexed struct {
	queries noteport.VaultHolds
}

func NewIsVaultIndexed(queries noteport.VaultHolds) *IsVaultIndexed {
	if queries == nil {
		queries = noIndex{}
	}
	return &IsVaultIndexed{queries: queries}
}

// Execute returns false for a build with no index and for an index error.
func (u *IsVaultIndexed) Execute(ctx context.Context, v domain.Vault) bool {
	if u == nil {
		return false
	}
	target, err := u.queries.Has(ctx, v.ID)
	return err == nil && target
}

// noIndex is the index of a build that has none: it holds no vault.
type noIndex struct{}

func (noIndex) Has(context.Context, domain.VaultID) (bool, error) { return false, nil }
