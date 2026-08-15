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
// A listener that is not keeping up is skipped rather than waited for: a window
// that has fallen behind will ask for what it needs when it catches up, and one
// slow listener must not hold the vault's own reindexing.
type audience struct {
	mu   sync.Mutex
	next int
	to   map[int]chan changed
}

func (a *audience) listen() (<-chan changed, func()) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.to == nil {
		a.to = map[int]chan changed{}
	}
	id := a.next
	a.next++
	line := make(chan changed, 8)
	a.to[id] = line

	return line, func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		if line, open := a.to[id]; open {
			delete(a.to, id)
			close(line)
		}
	}
}

func (a *audience) tell(what changed) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, line := range a.to {
		select {
		case line <- what:
		default:
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
			a.Listeners.tell(changed{paths: res.Changed()})

		case <-lost:
			// More arrived at once than could be followed. Reading the vault
			// again is the answer, and the client is told to ask again.
			if _, err := a.Scan(ctx, a.Vault); err != nil {
				a.Failed.Store(err.Error())
				continue
			}
			a.Listeners.tell(changed{reload: true})
		}
	}
}
