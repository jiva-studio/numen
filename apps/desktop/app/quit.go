package app

import (
	"context"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

// visibility is the window going out of sight and returning to it. Hiding
// leaves the page drawing and answering, so what only it holds is passed
// after the window is gone from the screen.
type visibility struct {
	hide func()
	show func()
}

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

// windowTracker tracks active open windows and returns whether a closing window is the last.
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

func (t *windowTracker) closeSecondaryWindow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.count > 1 {
		t.count--
		return true
	}
	return false
}

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

func getQuitBound() time.Duration { return container.FlushTimeout() }

// vaultFlush flushes the vault, whichever way the window is requested to close.
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

func (g *vaultFlush) isFlushed() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.isDone
}

func (g *vaultFlush) run(parent context.Context, this *flushAttempt) {
	defer close(this.ready)

	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), getQuitBound())
	defer cancel()
	resolved := g.flush(ctx)

	g.mu.Lock()
	defer g.mu.Unlock()
	this.isFlushed = resolved
	g.isDone = resolved
	g.turn = nil
}
