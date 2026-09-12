package main

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
)

// visibility is the window going out of sight and coming back into it. Hiding
// leaves the page drawing and answering, so what only it holds is handed over
// after the window is gone from the screen.
type visibility struct {
	hide func()
	show func()
}

// closeWindow is the window being asked to go, and answers with whether it may.
//
// The window goes out of sight first and the vault settles behind it. A page
// holding text a person has to answer for calls the close off, the window comes
// back, and the close is asked for again once they have answered. That wait is
// on a person and is not measured.
func closeWindow(
	ctx context.Context,
	g *going,
	s visibility,
	answered func(context.Context) bool,
	retry func(),
) bool {
	s.hide()
	//nolint:contextcheck // the settling runs when the caller's context is already over, under a bound of its own
	if g.wait() {
		return true
	}
	s.show()
	go func() {
		if answered(ctx) {
			retry()
		}
	}()
	return false
}

// requestQuit is a quit that did not come through the window, and answers with
// whether the application may go.
//
// It is answered on the thread the page is served on, so the window is hidden
// and the settling happens off it, and the quit is asked for again once it is
// over. A settling that ended with a question standing puts the window back and
// asks for nothing: the person is answering it, and that is the whole of what
// the goroutine left behind may do.
func requestQuit(g *going, s visibility, quit func()) bool {
	if g.isSettled() {
		return true
	}
	go func() {
		s.hide()
		if g.wait() {
			quit()
			return
		}
		s.show()
	}()
	return false
}

// quitBound is how long the window waits for a page that says nothing to write
// what only it holds. A vault being changed waits under the same bound.
const quitBound = editor.HandedOverIn

// going is the vault settling, whichever way the window is asked to go.
//
// A settling that ends with a question standing leaves the vault as it was, and
// the next ask begins another one.
type going struct {
	settle func(context.Context) bool

	mu   sync.Mutex
	turn *turn
	done bool
}

// turn is one settling, and what it answered.
type turn struct {
	done    chan struct{}
	settled bool
}

// wait settles the vault and answers with whether it did. A settling already
// running is joined and its answer shared.
func (g *going) wait() bool {
	g.mu.Lock()
	if g.done {
		g.mu.Unlock()
		return true
	}
	this := g.turn
	if this == nil {
		this = &turn{done: make(chan struct{})}
		g.turn = this
		go g.begin(this)
	}
	g.mu.Unlock()

	<-this.done
	return this.settled
}

// isSettled reports whether there is nothing left owed.
func (g *going) isSettled() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.done
}

func (g *going) begin(this *turn) {
	defer close(this.done)

	ctx, cancel := context.WithTimeout(context.Background(), quitBound)
	defer cancel()
	settled := g.settle(ctx)

	g.mu.Lock()
	defer g.mu.Unlock()
	this.settled = settled
	g.done = settled
	g.turn = nil
}
