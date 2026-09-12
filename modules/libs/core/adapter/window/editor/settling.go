package editor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
)

// shutting is what the window owes landing before anything is taken away: one
// settling runs at a time, and whether the window has settled to go.
//
// A swap and a window closing both settle, and the second to arrive is
// refused. A window that settled to go shows no other vault.
type shutting struct {
	mu    sync.Mutex
	busy  bool
	going bool
}

// alone takes the window for one settling, and says why it cannot be had.
func (s *shutting) alone() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch {
	case s.going:
		return errGoing
	case s.busy:
		return errSettling
	}
	s.busy = true
	return nil
}

// free gives the window back, the settling being over and the window staying.
func (s *shutting) free() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.busy = false
}

// over gives the window back from the settling that ends it. gone is what that
// settling came to.
func (s *shutting) over(gone bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.busy = false
	s.going = gone
}

// errGoing is a vault asked for in a window that has settled to go, and which
// shows no other vault.
var errGoing = errors.New("this window is going")

// errSettling is a vault asked for while the window is already settling what it
// owes.
var errSettling = errors.New("the window is settling what it owes")

// Settle is everything owed landing before anything is taken away. It is called
// while the window is still drawn, and calling it again is free. It answers
// false where a page is holding work a person is being asked about, and then
// nothing has been taken away and the vault is as it was.
//
// A vault being opened settles too, and the close that arrives while it is
// running is answered false: the window stays, and the next ask settles again.
func (o *Installation) Settle(ctx context.Context) bool {
	switch err := o.shutting.alone(); {
	case errors.Is(err, errGoing):
		return true
	case err != nil:
		return false
	}

	settled := settle(ctx, o.API.Window, &o.API.Writing)
	o.shutting.over(settled)
	return settled
}

// WaitForAnswers waits for every page to write what it owes. It is what the
// window waits on while a person answers a question, and that wait is on a
// person and is not measured. It answers false where ctx ended or the vault was
// asked again.
func (o *Installation) WaitForAnswers(ctx context.Context) bool {
	return o.API.Window.WaitForAnswers(ctx)
}

// settle is everything owed landing: every client writes what only it holds,
// and then the writes already taken finish. It answers with whether the vault
// settled.
//
// A client that says nothing is bounded by ctx. A client raising a question
// ends the round: the vault and the door stay open, and the wait from there is
// on a person. The writes are not bounded.
func settle(ctx context.Context, pages *wire.Window, writes *inflight) bool {
	if !pages.Settling(ctx) {
		return false
	}
	<-writes.seal()
	return true
}

// HandedOverIn is how long a page has to write what only it holds when the
// vault it is drawing goes. A page raising a question has answered, and the
// wait from there is on a person.
const HandedOverIn = 3 * time.Second
