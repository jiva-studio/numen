package identity

import (
	"context"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// GetLinkWork tells the courier what to carry.
type GetLinkWork struct {
	links linkHold
	clock Clock
}

// NewGetLinkWork returns the use case that tells the courier what to carry.
func NewGetLinkWork(d Deps) *GetLinkWork {
	return &GetLinkWork{links: d.progress.links, clock: d.clock}
}

// Execute drops the blobs the version sinceVersion covers and returns the work: the channel to make or use, the blobs to post to the other side
// and whether the link ended. It returns at once when the version differs from sinceVersion; otherwise it waits up to wait, at most 20 seconds,
// for a change.
func (u *GetLinkWork) Execute(ctx context.Context, sinceVersion uint64, wait time.Duration) (identitydomain.LinkWork, error) {
	var timeout <-chan time.Time
	if wait > 0 {
		timeout = u.clock.After(min(wait, maxLinkWait))
	}
	for {
		work, changed := u.read(sinceVersion)
		if work.Version != sinceVersion || wait <= 0 {
			return work, nil
		}
		select {
		case <-changed:
		case <-timeout:
			work, _ = u.read(sinceVersion)
			return work, nil
		case <-ctx.Done():
			return identitydomain.LinkWork{}, ctx.Err()
		}
	}
}

func (u *GetLinkWork) read(sinceVersion uint64) (identitydomain.LinkWork, <-chan struct{}) {
	store, release := u.links.hold()
	defer release()
	store.expireLocked(u.clock.Now())
	link := store.current
	if link == nil {
		return identitydomain.LinkWork{Version: store.ticker.getVersion()}, store.ticker.getChanged()
	}
	link.outbox.Acknowledge(sinceVersion)
	return link.outbox.GetWork(store.ticker.getVersion()), store.ticker.getChanged()
}
