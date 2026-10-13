package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
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
			if err := vaults.indexVault(t.Context(), v); err != nil {
				t.Error(err)
			}
		}()
	}
	walks.Wait()

	// The opening is one, so the watch behind it is one.
	if len(vaults.opened) != 1 {
		t.Errorf("the window opened the vault %d times", len(vaults.opened))
	}
}

// A walk asked for once the window has begun closing is refused, so none begins
// after the index has been waited for.
func TestClosingWindowReadsNothing(t *testing.T) {
	_, _, vaults, _, held := makeWindow(t)

	vaults.closeAndWait()

	if err := vaults.indexVault(t.Context(), held[0]); !errors.Is(err, errClosing) {
		t.Errorf("a walk asked for while closing came back with %v", err)
	}
}

// A levelling asked for once the window has begun closing is refused, so none
// begins writing to the index after it has been waited for. The prose is on
// disk either way, and the window says the note is written and unlevelled.
func TestClosingWindowUpdatesNothing(t *testing.T) {
	_, _, vaults, _, held := makeWindow(t)

	vaults.closeAndWait()

	err := vaults.updateIndex(t.Context(), held[0], []string{"decks/Words.md"})
	if !errors.Is(err, errClosing) {
		t.Errorf("a levelling asked for while closing came back with %v", err)
	}
}

// A vault is levelled through the opening it was walked with, and neither opens
// the vault a second time.
func TestUpdateIndexUsesTheVaultsOwnOpenedVault(t *testing.T) {
	_, _, vaults, _, held := makeWindow(t)
	v := held[0]

	if err := vaults.indexVault(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if err := vaults.updateIndex(t.Context(), v, []string{"decks/Words.md"}); err != nil {
		t.Fatal(err)
	}
	if len(vaults.opened) != 1 {
		t.Errorf("levelling and walking opened the vault %d times", len(vaults.opened))
	}
}

// closeAndWait holds until the watch over an opened vault has ended.
func TestClosingHoldsUntilTheWatchHasEnded(t *testing.T) {
	cfg, db, _, _, held := makeWindow(t)
	lifetime, end := context.WithCancel(t.Context())
	defer end()
	vaults := &openVaults{
		cfg: cfg, db: db, lifetime: lifetime,
		recordVault: func(domain.Vault) {}, out: io.Discard,
	}
	if err := vaults.indexVault(t.Context(), held[0]); err != nil {
		t.Fatal(err)
	}

	closed := make(chan struct{})
	go func() {
		vaults.closeAndWait()
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("the window let go of the index while a watch was still running")
	case <-time.After(300 * time.Millisecond):
	}
	end()
	select {
	case <-closed:
	case <-time.After(10 * time.Second):
		t.Fatal("the window never let go once the watch ended")
	}
}

// What a watched vault changes under is reported to the window, through the
// opening that watches it.
func TestAChangeInAnOpenedVaultIsRecorded(t *testing.T) {
	_, _, vaults, _, held := makeWindow(t)
	v := held[0]
	var out syncBuffer
	vaults.out = &out
	told := make(chan domain.Vault, 8)
	vaults.recordVault = func(v domain.Vault) {
		select {
		case told <- v:
		default:
		}
	}
	if err := vaults.indexVault(t.Context(), v); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(v.Path, "decks", "Fresh.md"),
		[]byte("---\ntype: deck\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-told:
		if got.ID != v.ID {
			t.Errorf("the change was recorded against %q", got.ID)
		}
		if strings.Contains(out.String(), "numen-flashcards: One:") {
			t.Errorf("a change that went well was said as a fault: %q", out.String())
		}
	case <-time.After(20 * time.Second):
		t.Fatal("a change in the vault was not recorded")
	}
}

// A vault that is watched says nothing of not being followed, and one whose
// folder is not there says so.
func TestAnUnwatchedVaultIsSaidAndAWatchedOneIsNot(t *testing.T) {
	_, _, vaults, _, held := makeWindow(t)
	var out syncBuffer
	vaults.out = &out

	if err := vaults.indexVault(t.Context(), held[0]); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "is not being followed") {
		t.Errorf("a watched vault was said to be unwatched: %q", out.String())
	}

	missing := domain.Vault{ID: "missing", Name: "Missing", Path: filepath.Join(t.TempDir(), "nowhere")}
	vaults.getOpenedVault(missing)
	if want := "numen-flashcards: Missing is not being followed:"; !strings.Contains(out.String(), want) {
		t.Errorf("the log says %q", out.String())
	}
}

// A levelling that fails answers with what failed.
func TestUpdateIndexAnswersWithWhatFailed(t *testing.T) {
	_, _, vaults, _, held := makeWindow(t)
	ended, end := context.WithCancel(t.Context())
	end()

	if err := vaults.updateIndex(ended, held[0], []string{"decks/Words.md"}); err == nil {
		t.Error("a levelling that could not run answered with nothing")
	}
}

// What goes wrong following a vault is said with the vault's name.
func TestWhatGoesWrongFollowingAVaultIsSaid(t *testing.T) {
	_, db, vaults, _, held := makeWindow(t)
	v := held[0]
	var out syncBuffer
	vaults.out = &out
	if err := vaults.indexVault(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(v.Path, "decks", "Fresh.md"),
		[]byte("---\ntype: deck\n---\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(20 * time.Second)
	for !strings.Contains(out.String(), "numen-flashcards: One: ") {
		if time.Now().After(deadline) {
			t.Fatalf("nothing was said about an index that could not be written: %q", out.String())
		}
		time.Sleep(50 * time.Millisecond)
	}
}
