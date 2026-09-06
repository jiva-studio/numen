package main

import (
	"errors"
	"sync"
	"testing"
)

// A vault is walked once however many counts ask for it at the same moment.
func TestAVaultIsWalkedOnceHoweverManyAsk(t *testing.T) {
	_, _, vaults, _, held := built(t)
	v := held[0]

	var walks sync.WaitGroup
	for range 8 {
		walks.Add(1)
		go func() {
			defer walks.Done()
			if err := vaults.reads(t.Context(), v, func(int64) {}); err != nil {
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
	_, _, vaults, _, held := built(t)

	vaults.wait()

	if err := vaults.reads(t.Context(), held[0], func(int64) {}); !errors.Is(err, errGoing) {
		t.Errorf("a walk asked for while closing came back with %v", err)
	}
}

// A vault is levelled through the opening it was walked with, and neither opens
// the vault a second time.
func TestLevellingGoesThroughTheVaultsOwnOpening(t *testing.T) {
	_, _, vaults, _, held := built(t)
	v := held[0]

	if err := vaults.reads(t.Context(), v, func(int64) {}); err != nil {
		t.Fatal(err)
	}
	if err := vaults.level(t.Context(), v, []string{"decks/Words.md"}); err != nil {
		t.Fatal(err)
	}
	if len(vaults.openings) != 1 {
		t.Errorf("levelling and walking opened the vault %d times", len(vaults.openings))
	}
}
