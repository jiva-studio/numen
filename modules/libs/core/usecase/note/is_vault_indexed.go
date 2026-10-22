package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/noteport"
)

// IsVaultIndexed says whether the index already carries a vault.
type IsVaultIndexed struct {
	queries noteport.NoteQueries
}

func NewIsVaultIndexed(queries noteport.NoteQueries) *IsVaultIndexed {
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
