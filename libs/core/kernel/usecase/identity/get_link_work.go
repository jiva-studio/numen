package identity

import (
	"context"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// GetLinkWork tells the courier what to carry.
type GetLinkWork struct{ deps Deps }

// NewGetLinkWork returns the use case that tells the courier what to carry.
func NewGetLinkWork(d Deps) *GetLinkWork { return &GetLinkWork{deps: d} }

// Execute drops the blobs the version sinceVersion covers and returns the work: the channel to make or use, the blobs to post to the other side
// and whether the link ended. It returns at once when the version differs from sinceVersion; otherwise it waits up to wait, at most 20 seconds,
// for a change.
func (u *GetLinkWork) Execute(ctx context.Context, sinceVersion uint64, wait time.Duration) (identitydomain.LinkWork, error) {
	var timeout <-chan time.Time
	if wait > 0 {
		timeout = u.deps.clock.After(min(wait, maxLinkWait))
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
	store := u.deps.links
	store.mu.Lock()
	defer store.mu.Unlock()
	store.expireLocked(u.deps.clock.Now())
	work := identitydomain.LinkWork{Version: store.version}
	link := store.current
	if link == nil {
		return work, store.changed
	}
	link.acknowledge(sinceVersion)
	work.IsActive = !link.relay.isClosed || len(link.relay.items) > 0
	work.WantsChannel = link.relay.wantsChannel
	work.ChannelID = append([]byte(nil), link.relay.channelID...)
	work.Server = link.relay.server
	work.IsClosed = link.relay.isClosed
	for _, item := range link.relay.items {
		work.Outbound = append(work.Outbound, append([]byte(nil), item.blob...))
	}
	return work, store.changed
}
