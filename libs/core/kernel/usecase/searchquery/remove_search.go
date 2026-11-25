package searchquery

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
)

// DeleteSearch releases a search the caller started. The search is canceled and forgotten when no other
// caller that started an equal search still holds it.
type DeleteSearch struct {
	kept keptSearches
}

// Execute returns searchrun.ErrNotFound for a search that is unknown, expired, removed or of another plugin.
func (u *DeleteSearch) Execute(_ context.Context, owner string, id searchrun.ID) error {
	if _, err := u.kept.Find(owner, id); err != nil {
		return err
	}
	remaining := 0
	u.kept.Parts().Store.ChangeSearch(id, func(r *searchrun.Record) {
		r.Holders--
		remaining = r.Holders
	})
	if remaining > 0 {
		return nil
	}
	u.kept.Drop([]searchrun.ID{id})
	return nil
}
