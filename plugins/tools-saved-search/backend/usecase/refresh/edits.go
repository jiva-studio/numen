package refresh

import (
	"context"
	"sync"
)

// edits are the checks and runs started by the latest edit of each note. A new edit of a note cancels the
// one before it, and the end of the last watch of a note cancels its run.
type edits struct {
	mu      sync.Mutex
	next    int
	current map[watchKey]edit
}

type edit struct {
	id     int
	cancel context.CancelFunc
	isRun  bool
}

func newEdits() *edits {
	return &edits{mu: sync.Mutex{}, next: 0, current: map[watchKey]edit{}}
}

func (e *edits) start(ctx context.Context, key watchKey, isRun bool) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	defer e.mu.Unlock()
	if previous, isFound := e.current[key]; isFound {
		previous.cancel()
	}
	e.next++
	id := e.next
	e.current[key] = edit{id: id, cancel: cancel, isRun: isRun}
	return ctx, func() {
		e.mu.Lock()
		if e.current[key].id == id {
			delete(e.current, key)
		}
		e.mu.Unlock()
		cancel()
	}
}

func (e *edits) cancelRun(key watchKey) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if one, isFound := e.current[key]; isFound && one.isRun {
		one.cancel()
		delete(e.current, key)
	}
}
