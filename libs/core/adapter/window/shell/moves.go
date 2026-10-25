package shell

import (
	"context"
	"sync"
)

// changeSubscribers is everyone listening for something moving underneath the window.
//
// A listener that is busy is passed over: every message says the same thing, and one lost is one the
// next says over.
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
	pos := f.next
	f.next++
	f.subscribers[pos] = line
	return line, func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		if target, is := f.subscribers[pos]; is {
			delete(f.subscribers, pos)
			close(target)
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
// What changed is not carried and not acted on. Whoever draws asks what the vaults come to now, which
// is the one answer that cannot go stale.
func (a *API) ReportChange() { a.subscribers.notify() }

// ReportChangesFrom says ReportChange for everything one channel reports, until it closes or ctx is
// done.
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

// WatchMoves tells that something moved, until ctx is done. It is what the plugins of the installation
// are told through the host.
func (a *API) WatchMoves(ctx context.Context) (<-chan struct{}, error) {
	line, isDone := a.subscribers.subscribe()
	out := make(chan struct{}, 1)
	go func() {
		defer close(out)
		defer isDone()
		for {
			select {
			case <-ctx.Done():
				return
			case _, open := <-line:
				if !open {
					return
				}
				select {
				case out <- struct{}{}:
				default:
				}
			}
		}
	}()
	return out, nil
}
