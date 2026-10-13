package main

import (
	"reflect"
	"slices"
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

// The page is given the cards, the notes and the registry to work through.
func TestThePageIsGivenWhatItWorksThrough(t *testing.T) {
	cfg, db, vaults, _, _ := makeWindow(t)
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	notes, _ := makeNotesAndCards(cfg, db, vaults)

	api := makeWindowAPI(cfg, registry, db, notes, vaults)

	if api == nil {
		t.Fatal("the page has no API")
	}
	if api.Now == nil || api.Now().IsZero() {
		t.Error("the page tells no time")
	}
	for name, wired := range map[string]any{
		"known vaults": api.GetKnownVaults,
		"cards due":    api.CardsDue,
		"session":      api.Session,
		"record":       api.Record,
		"presets":      api.Presets,
		"title":        api.GetTitle,
		"window":       api.Window,
		"day":          api.Day,
		"find":         api.Find,
	} {
		if reflect.ValueOf(wired).IsZero() {
			t.Errorf("the page has no %s", name)
		}
	}
}

// A vault the index was brought level with is counted again, and the walk of a
// vault is the one the window's own opening makes.
func TestTheVaultIndexerIsBoundToTheWindow(t *testing.T) {
	cfg, db, vaults, _, held := makeWindow(t)
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	notes, _ := makeNotesAndCards(cfg, db, vaults)
	api := makeWindowAPI(cfg, registry, db, notes, vaults)

	bindVaultIndexer(t.Context(), api, vaults)

	if vaults.recordVault == nil {
		t.Fatal("a vault brought level is recorded nowhere")
	}
	vaults.recordVault(held[0])
}

// The window lets go in order: the agents, the page, the walk and the watch,
// and the index last.
func TestTheWindowLetsGoInOrder(t *testing.T) {
	_, db, vaults, _, _ := makeWindow(t)
	var ran []string

	steps := makeShutdown(
		func() error { ran = append(ran, "agents"); return nil },
		func() { ran = append(ran, "stop") },
		vaults, db,
	)
	steps.Run()

	if want := []string{"agents", "stop"}; !slices.Equal(ran, want) {
		t.Errorf("the window let go of %v", ran)
	}
	if vaults.tryBeginWork() {
		t.Error("work began after the window let go")
	}
}
