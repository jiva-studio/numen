package webui

import (
	"context"
	"sync"
)

// changed is what a client is told: the notes that are different now, or that
// the vault has to be read again.
type changed struct {
	paths  []string
	reload bool
}

// audience is everyone listening for changes.
//
// A listener that is not keeping up is not waited for: one slow listener must
// not hold up the vault's own reindexing. It is not skipped either — a message
// that never arrives leaves a client showing something stale and certain it is
// current. It is told to ask for everything again, which is the same answer the
// watcher gives when more arrives at once than it can follow.
type audience struct {
	mu   sync.Mutex
	next int
	to   map[int]*line
}

// line is one listener, and whether it is owed a message it never received.
type line struct {
	ch     chan changed
	behind bool
}

func (a *audience) listen() (<-chan changed, func()) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.to == nil {
		a.to = map[int]*line{}
	}
	id := a.next
	a.next++
	l := &line{ch: make(chan changed, 8)}
	a.to[id] = l

	return l.ch, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		if l, open := a.to[id]; open {
			delete(a.to, id)
			close(l.ch)
		}
	}
}

func (a *audience) tell(what changed) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, l := range a.to {
		message := what
		if l.behind {
			// Owed one already: whatever it is now behind on, reading the lot
			// again covers it.
			message = changed{reload: true}
		}
		select {
		case l.ch <- message:
			l.behind = false
		default:
			l.behind = true
		}
	}
}

// follow keeps the index level with the vault and tells everyone listening
// which notes moved.
func (a *API) follow(ctx context.Context, changes <-chan []string, lost <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return

		case paths, open := <-changes:
			if !open {
				return
			}
			res, err := a.Index(ctx, a.Vault, paths)
			if err != nil {
				a.Failed.Store(err.Error())
				continue
			}
			// What failed before and works now is no longer failing. A message
			// left in place outlives what it described.
			a.Failed.Store("")
			a.Listeners.tell(changed{paths: res.Changed()})

		case <-lost:
			// More arrived at once than could be followed. Reading the vault
			// again is the answer, and the client is told to ask again.
			if _, err := a.Scan(ctx, a.Vault); err != nil {
				a.Failed.Store(err.Error())
				continue
			}
			a.Failed.Store("")
			a.Listeners.tell(changed{reload: true})
		}
	}
}
