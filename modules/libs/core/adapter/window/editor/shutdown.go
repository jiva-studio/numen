package editor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
)

// shutdownState is whether a flush of pending writes is running and whether the
// window has closed. Vault swaps and window close both flush, one at a time.
//
// A window that has closed shows no other vault.
type shutdownState struct {
	mu        sync.Mutex
	isBusy    bool
	isClosing bool
}

// tryBegin starts a flush, or returns the reason one cannot start.
func (s *shutdownState) tryBegin() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case s.isClosing:
		return errWindowClosing
	case s.isBusy:
		return errShutdownInProgress
	}
	s.isBusy = true
	return nil
}

// endKeepOpen ends the flush and leaves the window open.
func (s *shutdownState) endKeepOpen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isBusy = false
}

// end ends the flush. closed is whether the window is now closed.
func (s *shutdownState) end(closed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isBusy = false
	s.isClosing = closed
}

// errWindowClosing is the error for a vault requested in a closed window.
var errWindowClosing = errors.New("this window is closing")

// errShutdownInProgress is the error for a vault requested while a flush is
// running.
var errShutdownInProgress = errors.New("the window is flushing pending writes")

// FlushPending has every page write what only it holds, then waits for the
// writes already taken. It is called while the window is still showing, and
// calling it again returns true at once. It returns false where a page holds
// work a person has to answer for, and then nothing has been taken away.
//
// A vault swap flushes too. A close that arrives while a flush runs returns
// false: the window stays, and the next close flushes again.
func (o *Installation) FlushPending(ctx context.Context) bool {
	switch err := o.shutdown.tryBegin(); {
	case errors.Is(err, errWindowClosing):
		return true
	case err != nil:
		return false
	}

	flushed := flushPending(ctx, o.State.Window, &o.State.Writing)
	o.shutdown.end(flushed)
	return flushed
}

// WaitForFlush waits for every page to write its pending data. The wait is on
// ctx ended or the vault was requested again.
func (o *Installation) WaitForFlush(ctx context.Context) bool {
	return o.State.Window.WaitForFlush(ctx)
}

// flushPending has every client write what only it holds, then waits for the
// writes already taken. It returns whether the flush completed.
//
// A silent client is bounded by ctx. A client raising a question ends the
// flush with false: the vault stays open and the wait is on a person. The
// writes themselves have no bound.
func flushPending(ctx context.Context, pages *wire.Window, writes *inflight) bool {
	if !pages.RequestFlush(ctx) {
		return false
	}
	<-writes.seal()
	return true
}

// FlushTimeout is how long a page has to write what only it holds when the
// vault goes. A page raising a question has answered, and the wait from there
// is on a person.
const FlushTimeout = 3 * time.Second
