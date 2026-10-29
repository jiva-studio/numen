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

// closeWindow reports whether the window may close.
//
// The window is hidden first and the vault is flushed in the background. A page
// holding unsaved text that needs a user decision cancels the close, the window
// is shown again, and the close is requested again after the user has answered.
// That wait depends on the user and has no time bound.
func closeWindow(
	ctx context.Context,
	g *vaultFlush,
	s visibility,
	confirm func(context.Context) bool,
	retry func(),
) bool {
	s.hide()
	//nolint:contextcheck // the settling runs when the caller's context is already over, under a bound of its own
	if g.flushVault() {
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
// before quitting; secondary windows close without flushing.
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

// requestQuit handles a quit that did not come through a window and reports
// whether the application may exit.
//
// It runs on the thread that serves the page, so the window is hidden and the
// flush runs in a goroutine, and the quit is requested again when the flush
// ends. A flush that ended awaiting a user decision shows the window again and
// does not request the quit; the quit follows after the user has answered.
func requestQuit(g *vaultFlush, s visibility, quit func()) bool {
	if g.isFlushed() {
		return true
	}
	go func() {
		s.hide()
		if g.flushVault() {
			quit()
			return
		}
		s.show()
	}()
	return false
}

// quitBound is how long the window waits for a page that does not respond to
// write the work only it holds. A vault being changed waits for the same bound.
const quitBound = editor.FlushTimeout

// vaultFlush flushes the vault, whichever way the window is asked to close.
//
// A flush that ends awaiting a user decision leaves the vault unchanged, and
// the next request starts another flush.
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
// running is joined and its answer shared.
func (g *vaultFlush) flushVault() bool {
	g.mu.Lock()
	if g.isDone {
		g.mu.Unlock()
		return true
	}
	this := g.turn
	if this == nil {
		this = &flushAttempt{ready: make(chan struct{})}
		g.turn = this
		go g.run(this)
	}
	g.mu.Unlock()

	<-this.ready
	return this.isFlushed
}

// isFlushed reports whether the vault has no unwritten work.
func (g *vaultFlush) isFlushed() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.isDone
}

func (g *vaultFlush) run(this *flushAttempt) {
	defer close(this.ready)

	ctx, cancel := context.WithTimeout(context.Background(), quitBound)
	defer cancel()
	resolved := g.flush(ctx)

	g.mu.Lock()
	defer g.mu.Unlock()
	this.isFlushed = resolved
	g.isDone = resolved
	g.turn = nil
}
