package vaultwatch

import (
	"context"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
)

type debounceState struct {
	pending map[string]bool
	ready   []string
	inReady map[string]bool
	hold    <-chan time.Time
	queued  []walkRequest
	again   []string
	recheck <-chan time.Time
}

func newDebounceState() *debounceState {
	return &debounceState{
		pending: make(map[string]bool),
		ready:   nil,
		inReady: make(map[string]bool),
		hold:    nil,
		queued:  nil,
		again:   nil,
		recheck: nil,
	}
}

func (s *debounceState) reset() {
	clear(s.pending)
	clear(s.inReady)
	s.ready = nil
	s.hold = nil
	s.queued = nil
}

func (s *debounceState) rescan(lostChan chan<- struct{}) {
	s.reset()
	select {
	case lostChan <- struct{}{}:
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

func (s *debounceState) walkChan(requested chan<- walkRequest) (chan<- walkRequest, walkRequest) {
	if len(s.queued) > 0 {
		return requested, s.queued[0]
	}
	return nil, walkRequest{}
}

func (s *debounceState) handleWalkResult(one walkResult, delay time.Duration, droppedChan chan<- struct{}) {
	if one.isWholeVault {
		s.rescan(droppedChan)
		return
	}
	if one.isSecond {
		for _, path := range one.paths {
			s.pending[path] = true
		}
		s.startHold(delay)
		return
	}
	s.again = append(s.again, one.folder)
	if s.recheck == nil {
		s.recheck = time.After(delay)
	}
	// A folder holding no file adds no source to report, yet the folder is a change.
	if len(one.paths) == 0 {
		s.rescan(droppedChan)
		return
	}
	for _, path := range one.paths {
		s.pending[path] = true
	}
	s.startHold(delay)
}

// queueRecheck asks for the folders walked once to be walked again.
func (s *debounceState) queueRecheck() {
	s.recheck = nil
	for _, folder := range s.again {
		s.queued = append(s.queued, walkRequest{folder: folder, isSecond: true})
	}
	s.again = nil
}

func (s *debounceState) handleWaitingEvents(shape *vaultreader.Folders, waiting *queue, delay time.Duration, dropped chan<- struct{}) bool {
	events, finished, isDone := waiting.take()
	if finished {
		s.rescan(dropped)
		if err := shape.ReloadShape(); err != nil {
			// The shape lacks a folder the walk could not enter, so the vault is read again.
			s.rescan(dropped)
		}
	}
	for _, event := range events {
		paths, whole, folder := shape.GetConcernedPaths(event.Path())
		if whole {
			s.rescan(dropped)
			continue
		}
		if folder != "" {
			s.queued = append(s.queued, walkRequest{folder: folder})
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

func debounce(
	ctx context.Context,
	shape *vaultreader.Folders,
	opts vaultconfig.Options,
	waiting *queue,
	changes chan<- []string,
	disconnected chan<- struct{},
) {
	state := newDebounceState()
	request := make(chan walkRequest)
	walked := make(chan walkResult)

	going := make(chan struct{})
	var walking sync.WaitGroup
	walking.Go(func() {
		walkFolders(ctx, shape, going, request, walked)
	})
	defer walking.Wait()
	defer close(going)

	delay := opts.GetDebounceDelay()
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
			state.handleWalkResult(one, delay, disconnected)

		case <-waiting.woke:
			if state.handleWaitingEvents(shape, waiting, delay, disconnected) {
				return
			}

		case <-state.recheck:
			state.queueRecheck()

		case <-state.hold:
			state.flushPendingToReady()
		}
	}
}

// walkRequest names a folder new to the watch to walk. The second walk of a
// folder is the one made once the watch on it is in place.
type walkRequest struct {
	folder   string
	isSecond bool
}

// walkResult is what a walk of one folder new to the watch came to.
type walkResult struct {
	folder       string
	isSecond     bool
	paths        []string
	isWholeVault bool
}

func walkFolders(
	ctx context.Context,
	shape *vaultreader.Folders,
	going <-chan struct{},
	requested <-chan walkRequest,
	walked chan<- walkResult,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-going:
			return
		case req := <-requested:
			paths, whole := shape.GetPathsUnder(ctx, req.folder)
			select {
			case <-ctx.Done():
				return
			case <-going:
				return
			case walked <- walkResult{folder: req.folder, isSecond: req.isSecond, paths: paths, isWholeVault: whole}:
			}
		}
	}
}
