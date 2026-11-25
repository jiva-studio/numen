package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// GetLink shows the link in progress.
type GetLink struct {
	links linkHold
	clock Clock
}

// NewGetLink returns the use case that shows the link in progress.
func NewGetLink(d Deps) *GetLink { return &GetLink{links: d.progress.links, clock: d.getClock()} }

// Execute returns the link, or ErrNoLink when none is in progress or the person canceled it. A link that ran for ten minutes ends here.
func (u *GetLink) Execute(_ context.Context) (identitydomain.Link, error) {
	store, release := u.links.hold()
	defer release()
	store.expireLocked(u.clock.Now())
	link, err := store.getCurrentLocked()
	if err != nil {
		return identitydomain.Link{}, err
	}
	return link.progress.GetLink(), nil
}
