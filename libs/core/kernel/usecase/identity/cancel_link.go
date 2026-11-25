package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// CancelLink ends the link at the person's request.
type CancelLink struct {
	links linkHold
	clock machineport.InstallationClock
}

// NewCancelLink returns the use case that cancels the link.
func NewCancelLink(d Deps) *CancelLink { return &CancelLink{links: d.progress.links, clock: d.clock} }

// Execute tells the other side, closes the channel and forgets the link. A link that already ended is forgotten without a word;
// ErrNoLink means there was none.
func (u *CancelLink) Execute(_ context.Context) error {
	store, release := u.links.hold()
	defer release()
	store.expireLocked(u.clock.Now())
	link, err := store.getCurrentLocked()
	if err != nil {
		return err
	}
	if link.isEnded() {
		store.current = nil
		store.bumpLocked()
		return nil
	}
	store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonCanceled))
	store.endLocked(link, identitydomain.LinkPhaseCanceled, identitydomain.LinkReasonCanceled)
	return nil
}
