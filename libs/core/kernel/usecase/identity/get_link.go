package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// GetLink shows the link in progress.
type GetLink struct{ deps Deps }

// NewGetLink returns the use case that shows the link in progress.
func NewGetLink(d Deps) *GetLink { return &GetLink{deps: d} }

// Execute returns the link, or ErrNoLink when none is in progress or the person canceled it. A link that ran for ten minutes ends here.
func (u *GetLink) Execute(_ context.Context) (identitydomain.Link, error) {
	store := u.deps.links
	store.mu.Lock()
	defer store.mu.Unlock()
	store.expireLocked(u.deps.clock.Now())
	link, err := store.getCurrentLocked()
	if err != nil {
		return identitydomain.Link{}, err
	}
	return link.view(), nil
}
