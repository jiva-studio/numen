package main

import (
	"reflect"
	"testing"
)

// The notes and the cards the page and the tools are served are built, not
// empty.
func TestTheNotesAndCardsAreBuilt(t *testing.T) {
	cfg, db, vaults, _, _ := makeWindow(t)

	notes, cards := makeNotesAndCards(cfg, db, vaults)

	if reflect.ValueOf(notes).IsZero() {
		t.Error("no notes were built")
	}
	if reflect.ValueOf(cards).IsZero() {
		t.Error("no cards were built")
	}
}
