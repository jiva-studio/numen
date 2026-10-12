package flashcards

import (
	"context"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
)

// changeSubscribers is everyone listening for something moving underneath the window.
//
// A listener that is busy is passed over: every message says the same thing,
// and one lost is one the next says over.
type changeSubscribers struct {
	mu          sync.Mutex
	next        int
	subscribers map[int]chan struct{}
}

func (f *changeSubscribers) subscribe() (<-chan struct{}, func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.subscribers == nil {
		f.subscribers = make(map[int]chan struct{})
	}
	line := make(chan struct{}, 1)
	at := f.next
	// gomutants:disable-next-line INCREMENT_DECREMENT reason="a counter that steps down hands out distinct keys like one that steps up"
	f.next++
	f.subscribers[at] = line
	return line, func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		if held, is := f.subscribers[at]; is {
			delete(f.subscribers, at)
			close(held)
		}
	}
}

func (f *changeSubscribers) notify() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, line := range f.subscribers {
		select {
		case line <- struct{}{}:
		default:
		}
	}
}

// ReportChange says that something the window draws from has changed.
//
// What changed is not carried and not acted on. The page asks what the vaults
// come to now, which is the one answer that cannot go stale. Who watches what
// is the command's: this holds the listeners and nothing else.
func (a *API) ReportChange() {
	a.CardsDue.CardFaces().InvalidateAll()
	a.subscribers.notify()
}

// ReportChangesFrom says ReportChange for everything one channel reports, until it closes
// or ctx is done.
func (a *API) ReportChangesFrom(ctx context.Context, moved <-chan struct{}) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, open := <-moved:
				if !open {
					return
				}
				a.ReportChange()
			}
		}
	}()
}

// WatchReloads says something moved, for as long as the caller listens.
//
// What moved is not carried: what this window shows is counts and the cards
// behind them, and they are asked for again whatever changed.
func (a *API) WatchReloads(
	ctx context.Context,
	_ *connect.Request[v1.WatchReloadsRequest],
	out *connect.ServerStream[v1.WatchReloadsResponse],
) error {
	line, done := a.subscribers.subscribe()
	defer done()

	// Named as listening before anything has moved. A stream that says nothing
	// until a file changes cannot be told from one that never opened.
	if err := out.Send(&v1.WatchReloadsResponse{}); err != nil {
		return err
	}

	repeat := time.NewTicker(wire.Again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			// Nothing moved, which is what the opening message says too. A page
			// that has gone fails the write.
			if err := out.Send(&v1.WatchReloadsResponse{}); err != nil {
				return err
			}
		case <-line:
			if err := out.Send(&v1.WatchReloadsResponse{Reload: true}); err != nil {
				return err
			}
		}
	}
}
