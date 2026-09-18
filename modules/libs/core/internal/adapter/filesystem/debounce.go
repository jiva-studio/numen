package filesystem

import (
	"context"
	"sync"
	"time"
)

// debounce collects events for a hold and reports each path once.
//
// Reading the events and delivering them are kept apart. Whoever listens takes
// as long as it takes to refresh what it was told about, and the operating
// system goes on producing events meanwhile.
func debounce(
	ctx context.Context,
	shape *folders,
	opts Options,
	waiting *queue,
	changes chan<- []string,
	lost chan<- struct{},
) {
	pending := map[string]bool{}
	// ready is what the hold has already closed over, waiting to be taken. It
	// keeps growing while nobody takes it: one batch, never two.
	var ready []string
	inReady := map[string]bool{}
	var hold <-chan time.Time
	// queued is the folders new to the watch that are waiting to be walked, and
	// walked is what a walk of one came to. A walk made here would stop the
	// backlog emptying for as long as it took, which is the one thing this
	// goroutine may never do.
	var queued []string
	asked := make(chan string)
	walked := make(chan found)

	going := make(chan struct{})
	var walking sync.WaitGroup
	walking.Add(1)
	go func() {
		defer walking.Done()
		walks(ctx, shape, going, asked, walked)
	}()
	defer walking.Wait()
	defer close(going)

	forget := func() {
		clear(pending)
		clear(inReady)
		ready, hold, queued = nil, nil, nil
	}

	rescan := func() {
		forget()
		select {
		case lost <- struct{}{}:
		default:
		}
	}

	for {
		// The delivery arm is only in the select when there is something to
		// deliver; a nil channel is never chosen. The walking arm the same.
		var out chan<- []string
		if len(ready) > 0 {
			out = changes
		}
		var walk chan<- string
		var next string
		if len(queued) > 0 {
			walk, next = asked, queued[0]
		}

		select {
		case <-ctx.Done():
			return

		case out <- ready:
			ready = nil
			clear(inReady)

		case walk <- next:
			queued = queued[1:]

		case one := <-walked:
			if one.isWholeVault {
				rescan()
				continue
			}
			for _, path := range one.paths {
				pending[path] = true
			}
			if hold == nil && len(pending) > 0 {
				hold = time.After(opts.hold())
			}

		case <-waiting.woke:
			events, over, done := waiting.take()
			if over {
				// The backlog filled and what stood in it was dropped. Which
				// paths those were is unknowable, so the vault is read again,
				// and the shape with it: an event that made a folder is among
				// what went.
				rescan()
				shape.rereadShape()
			}
			for _, event := range events {
				paths, whole, folder := shape.concerns(event.Path())
				if whole {
					// A folder that is gone takes sources with it, and their
					// paths are known only to the index.
					rescan()
					continue
				}
				if folder != "" {
					queued = append(queued, folder)
				}
				for _, path := range paths {
					pending[path] = true
				}
			}
			// Measured from the first event of a batch, not the last. A vault
			// being written to continuously — a sync client, a checkout — never
			// stops long enough for a deadline that moves with it.
			if hold == nil && len(pending) > 0 {
				hold = time.After(opts.hold())
			}
			if done {
				return
			}

		case <-hold:
			hold = nil
			for path := range pending {
				if !inReady[path] {
					inReady[path] = true
					ready = append(ready, path)
				}
			}
			clear(pending)
		}
	}
}

// found is what a walk of one folder new to the watch came to.
type found struct {
	paths        []string
	isWholeVault bool
}

// walks walks the folders handed to it, one after another, and answers with
// what each holds. It stands beside the debounce, so a folder that arrives with
// a thousand files in it is walked while the backlog goes on emptying.
func walks(
	ctx context.Context,
	shape *folders,
	going <-chan struct{},
	asked <-chan string,
	walked chan<- found,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-going:
			return
		case at := <-asked:
			paths, whole := shape.getPathsUnder(ctx, at)
			select {
			case <-ctx.Done():
				return
			case <-going:
				return
			case walked <- found{paths: paths, isWholeVault: whole}:
			}
		}
	}
}
