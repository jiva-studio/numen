package shell

import (
	"context"
	"sync"
)

// changeSubscribers is the set of listeners notified when the window's source data changes.
//
// A busy listener is skipped: every message carries the same signal, and a
// dropped message is covered by the next one.
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

// ReportChange reports that data the window renders from has changed.
//
// The change itself is not carried. Receivers query the current state of the
// vaults, which cannot be stale.
func (a *API) ReportChange() { a.subscribers.notify() }

// ReportChangesFrom calls ReportChange for every signal on the channel, until
// it closes or ctx is done.
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

// WatchMoves signals that something moved, until ctx is done. Plugins of the
// installation receive it through the host.
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
