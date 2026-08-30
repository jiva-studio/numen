package main

import (
	"testing"
	"time"
)

// What moved is not carried past the window's door: the page asks what the
// vaults come to whatever it was, so a channel of paths becomes a channel of
// nothing.
func TestWhatMovedIsNotCarriedPast(t *testing.T) {
	paths := make(chan []string, 1)
	moved := drop(paths)

	paths <- []string{"decks/Words.md"}
	select {
	case <-moved:
	case <-time.After(5 * time.Second):
		t.Fatal("a deck was written and nothing was said")
	}
}

// The watcher stopping stops the window's side of it, so a vault that is no
// longer followed leaves nothing running behind it.
func TestNothingIsSaidOnceTheWatcherStops(t *testing.T) {
	paths := make(chan []string)
	moved := drop(paths)

	close(paths)
	select {
	case _, open := <-moved:
		if open {
			t.Error("the watcher stopped and something was still said")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the watcher stopped and the window is still listening")
	}
}
