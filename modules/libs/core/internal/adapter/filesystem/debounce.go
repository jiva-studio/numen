package filesystem

import (
	"context"
	"sync"
	"time"
)

type debounceState struct {
	pending map[string]bool
	ready   []string
	inReady map[string]bool
	hold    <-chan time.Time
	queued  []string
}

func newDebounceState() *debounceState {
	return &debounceState{
		pending: make(map[string]bool),
		ready:   nil,
		inReady: make(map[string]bool),
		hold:    nil,
		queued:  nil,
	}
}

func (s *debounceState) reset() {
	clear(s.pending)
	clear(s.inReady)
	s.ready = nil
	s.hold = nil
	s.queued = nil
}

func (s *debounceState) rescan(lost chan<- struct{}) {
	s.reset()
	select {
	case lost <- struct{}{}:
	default:
	}
}

func (s *debounceState) startHold(d time.Duration) {
	if s.hold == nil && len(s.pending) > 0 {
		s.hold = time.After(d)
	}
}

func (s *debounceState) deliverChan(changes chan<- []string) chan<- []string {
	if len(s.ready) > 0 {
		return changes
	}
	return nil
}

func (s *debounceState) walkChan(asked chan<- string) (chan<- string, string) {
	if len(s.queued) > 0 {
		return asked, s.queued[0]
	}
	return nil, ""
}

func (s *debounceState) handleWalkResult(one walkResult, delay time.Duration, lost chan<- struct{}) {
	if one.isWholeVault {
		s.rescan(lost)
		return
	}
	for _, path := range one.paths {
		s.pending[path] = true
	}
	s.startHold(delay)
}

func (s *debounceState) handleWaitingEvents(shape *folders, waiting *queue, delay time.Duration, lost chan<- struct{}) bool {
	events, over, isDone := waiting.take()
	if over {
		s.rescan(lost)
		shape.reloadShape()
	}
	for _, event := range events {
		paths, whole, folder := shape.getConcernedPaths(event.Path())
		if whole {
			s.rescan(lost)
			continue
		}
		if folder != "" {
			s.queued = append(s.queued, folder)
		}
		for _, path := range paths {
			s.pending[path] = true
		}
	}
	s.startHold(delay)
	return isDone
}

func (s *debounceState) flushPendingToReady() {
	s.hold = nil
	for path := range s.pending {
		if !s.inReady[path] {
			s.inReady[path] = true
			s.ready = append(s.ready, path)
		}
	}
	clear(s.pending)
}

func (s *debounceState) clearReady() {
	s.ready = nil
	clear(s.inReady)
}

func (s *debounceState) popQueued() {
	s.queued = s.queued[1:]
}

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
	state := newDebounceState()
	request := make(chan string)
	walked := make(chan walkResult)

	going := make(chan struct{})
	var walking sync.WaitGroup
	walking.Add(1)
	go func() {
		defer walking.Done()
		walkFolders(ctx, shape, going, request, walked)
	}()
	defer walking.Wait()
	defer close(going)

	delay := opts.getDebounceDelay()
	for {
		out := state.deliverChan(changes)
		walk, next := state.walkChan(request)

		select {
		case <-ctx.Done():
			return

		case out <- state.ready:
			state.clearReady()

		case walk <- next:
			state.popQueued()

		case one := <-walked:
			state.handleWalkResult(one, delay, lost)

		case <-waiting.woke:
			if state.handleWaitingEvents(shape, waiting, delay, lost) {
				return
			}

		case <-state.hold:
			state.flushPendingToReady()
		}
	}
}

// walkResult is what a walk of one folder new to the watch came to.
type walkResult struct {
	paths        []string
	isWholeVault bool
}

// walkFolders walkFolders the folders handed to it, one after another, and answers with
// what each holds. It stands beside the debounce, so a folder that arrives with
// a thousand files in it is walked while the backlog goes on emptying.
func walkFolders(
	ctx context.Context,
	shape *folders,
	going <-chan struct{},
	asked <-chan string,
	walked chan<- walkResult,
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
			case walked <- walkResult{paths: paths, isWholeVault: whole}:
			}
		}
	}
}
