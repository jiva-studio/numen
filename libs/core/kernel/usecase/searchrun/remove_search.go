package searchrun

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchkept"
)

// RemoveSearch cancels a running search and forgets it.
type RemoveSearch struct {
	kept *searchkept.Searches
}

// Execute returns searchrun.ErrNotFound for a search that is unknown, expired, removed or of another plugin.
func (u *RemoveSearch) Execute(_ context.Context, owner string, id searchrun.ID) error {
	if _, _, err := u.kept.Find(owner, id); err != nil {
		return err
	}
	u.kept.Drop([]searchrun.ID{id})
	return nil
}
