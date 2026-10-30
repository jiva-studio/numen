package app

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"
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
	g *vaultFlush,
	s visibility,
	confirm func(context.Context) bool,
	retry func(),
) bool {
	s.hide()
	if g.flushVault(ctx) {
		return true
	}
	s.show()
	go func() {
		if confirm(ctx) {
			retry()
		}
	}()
	return false
}

// windowTracker tracks active open windows and answers whether a closing window is the last.
type windowTracker struct {
	mu    sync.Mutex
	count int
}

func newWindowTracker() *windowTracker {
	return &windowTracker{}
}

func (t *windowTracker) addWindow() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.count++
}

func (t *windowTracker) removeWindow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.count > 0 {
		t.count--
	}
	return t.count == 0
}

func (t *windowTracker) isLast() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.count <= 1
}

func (t *windowTracker) countWindows() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.count
}

// closeSecondaryWindow decrements the window count if there is more than one window.
// It reports true if this was a secondary window (count > 1) and count was decremented,
// or false if this is the final window and closing must flush the vault.
func (t *windowTracker) closeSecondaryWindow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.count > 1 {
		t.count--
		return true
	}
	return false
}

// handleWindowClosing handles a window close event. The last window flushes the vault
// before quitting; secondary windows simply close.
func handleWindowClosing(
	ctx context.Context,
	tracker *windowTracker,
	g *vaultFlush,
	s visibility,
	confirm func(context.Context) bool,
	retry func(),
) bool {
	if tracker.closeSecondaryWindow() {
		return true
	}
	return closeWindow(ctx, g, s, confirm, retry)
}

// requestQuit is a quit that did not come through the window, and answers with
// whether the application may go.
//
// It is answered on the thread the page is served on, so the window is hidden
// and the settling happens off it, and the quit is asked for again once it is
// over. A settling that ended with a question standing puts the window back and
// asks for nothing: the person is answering it, and that is the whole of what
func requestQuit(g *vaultFlush, s visibility, quit func()) bool {
	if g.isFlushed() {
		return true
	}
	go func() {
		s.hide()
		if g.flushVault(context.Background()) {
			quit()
			return
		}
		s.show()
	}()
	return false
}

// quitBound is how long the window waits for a page that says nothing to write
// what only it holds. A vault being changed waits under the same bound.
const quitBound = editor.FlushTimeout

// vaultFlush is the vault settling, whichever way the window is asked to go.
//
// A settling that ends with a question standing leaves the vault as it was, and
// the next ask begins another one.
type vaultFlush struct {
	flush func(context.Context) bool

	mu     sync.Mutex
	turn   *flushAttempt
	isDone bool
}

// flushAttempt is one settling, and what it answered.
type flushAttempt struct {
	ready     chan struct{}
	isFlushed bool
}

// flushVault settles the vault and answers with whether it did. A settling already
// running is joined and its answer shared. The settling keeps the values of ctx and runs under
// quitBound, whether or not ctx is cancelled.
func (g *vaultFlush) flushVault(ctx context.Context) bool {
	g.mu.Lock()
	if g.isDone {
		g.mu.Unlock()
		return true
	}
	this := g.turn
	if this == nil {
		this = &flushAttempt{ready: make(chan struct{})}
		g.turn = this
		go g.run(ctx, this)
	}
	g.mu.Unlock()

	<-this.ready
	return this.isFlushed
}

// isFlushed reports whether there is nothing left owed.
func (g *vaultFlush) isFlushed() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.isDone
}

func (g *vaultFlush) run(parent context.Context, this *flushAttempt) {
	defer close(this.ready)

	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), quitBound)
	defer cancel()
	resolved := g.flush(ctx)

	g.mu.Lock()
	defer g.mu.Unlock()
	this.isFlushed = resolved
	g.isDone = resolved
	g.turn = nil
}
