package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// IsVaultIndexed says whether the index already carries a vault.
type IsVaultIndexed struct {
	queries port.NoteQueries
}

func NewIsVaultIndexed(queries port.NoteQueries) *IsVaultIndexed {
	return &IsVaultIndexed{queries: queries}
}

// Execute returns false for a build with no index and for an index error.
func (u *IsVaultIndexed) Execute(ctx context.Context, v domain.Vault) bool {
	if u == nil || u.queries == nil {
		return false
	}
	target, err := u.queries.Has(ctx, v.ID)
	return err == nil && target
}
