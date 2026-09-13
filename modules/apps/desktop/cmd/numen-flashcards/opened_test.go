package main

import (
	"errors"
	"sync"
	"testing"
)

// A vault is walked once however many counts ask for it at the same moment.
func TestAVaultIsWalkedOnceHoweverManyAsk(t *testing.T) {
	_, _, vaults, _, held := makeWindow(t)
	v := held[0]

	var walks sync.WaitGroup
	for range 8 {
		walks.Add(1)
		go func() {
			defer walks.Done()
			if err := vaults.readVault(t.Context(), v); err != nil {
				t.Error(err)
			}
		}()
	}
	walks.Wait()

	// The opening is one, so the watch behind it is one.
	if len(vaults.openings) != 1 {
		t.Errorf("the window opened the vault %d times", len(vaults.openings))
	}
}

// A walk asked for once the window has begun closing is refused, so none begins
// after the index has been waited for.
func TestAWindowThatIsGoingWalksNothing(t *testing.T) {
	_, _, vaults, _, held := makeWindow(t)

	vaults.wait()

	if err := vaults.readVault(t.Context(), held[0]); !errors.Is(err, errGoing) {
		t.Errorf("a walk asked for while closing came back with %v", err)
	}
}

// A levelling asked for once the window has begun closing is refused, so none
// begins writing to the index after it has been waited for. The prose is on
// disk either way, and the window says the note is written and unlevelled.
func TestAWindowThatIsGoingLevelsNothing(t *testing.T) {
	_, _, vaults, _, held := makeWindow(t)

	vaults.wait()

	err := vaults.level(t.Context(), held[0], []string{"decks/Words.md"})
	if !errors.Is(err, errGoing) {
		t.Errorf("a levelling asked for while closing came back with %v", err)
	}
}

// A vault is levelled through the opening it was walked with, and neither opens
// the vault a second time.
func TestLevellingGoesThroughTheVaultsOwnOpening(t *testing.T) {
	_, _, vaults, _, held := makeWindow(t)
	v := held[0]

	if err := vaults.readVault(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if err := vaults.level(t.Context(), v, []string{"decks/Words.md"}); err != nil {
		t.Fatal(err)
	}
	if len(vaults.openings) != 1 {
		t.Errorf("levelling and walking opened the vault %d times", len(vaults.openings))
	}
}
