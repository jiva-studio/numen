package searchrun

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
)

// RemoveSearch cancels a running search and forgets it.
type RemoveSearch struct {
	kept *Kept
}

// Execute returns searchrun.ErrNotFound for a search that is unknown, expired, removed or of another plugin.
func (u *RemoveSearch) Execute(_ context.Context, owner string, id searchrun.ID) error {
	if _, _, err := u.kept.find(owner, id); err != nil {
		return err
	}
	u.kept.drop([]searchrun.ID{id})
	return nil
}
