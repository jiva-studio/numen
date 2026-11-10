package windowsession

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
)

func TestShowingAnotherVaultStopsTheFirstAndStartsTheSecond(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())

	if err := w.scenarios.Switching.Show.Execute(t.Context(), studio(), nil); err != nil {
		t.Fatal(err)
	}

	want := []string{"recorded v-garden", "start v-garden", "flush", "retire v-garden", "recorded v-studio", "start v-studio"}
	if got := w.log.all(); !slices.Equal(got, want) {
		t.Errorf("the window did %v, want %v", got, want)
	}
	if got := w.rt.GetCurrentVault(); got.ID != studio().ID {
		t.Errorf("the window shows %q, want %q", got.ID, studio().ID)
	}
}

func TestShowVaultPublishesAReloadAndOpensWrites(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())

	if err := w.scenarios.Switching.Show.Execute(t.Context(), studio(), nil); err != nil {
		t.Fatal(err)
	}

	if got := w.changes.all(); len(got) != 1 || !got[0].ShouldReload {
		t.Errorf("the views were told %v, want one reload", got)
	}
	if w.writes.reopened != 1 || w.pages.canceled != 1 {
		t.Errorf("writes opened %d times and the flush ended %d times, want once each", w.writes.reopened, w.pages.canceled)
	}
}

func TestShowingTheVaultThatIsShownChangesNothing(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())

	if err := w.scenarios.Switching.Show.Execute(t.Context(), garden(), nil); err != nil {
		t.Fatal(err)
	}

	if got := w.passes.vaults(); len(got) != 1 || w.pages.flushes != 0 {
		t.Errorf("%d passes started and %d flushes ran, want the first pass alone", len(got), w.pages.flushes)
	}
}

func TestAPageAwaitingAnAnswerKeepsTheVaultInTheWindow(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())
	w.pages.isAsking = true

	err := w.scenarios.Switching.Show.Execute(t.Context(), studio(), nil)

	if !errors.Is(err, window.ErrAnswerPending) {
		t.Fatalf("the switch returned %v, want %v", err, window.ErrAnswerPending)
	}
	if got := w.rt.GetCurrentVault(); got.ID != garden().ID || w.passes.retired != 0 {
		t.Errorf("the window shows %q with %d passes retired, want %q untouched", got.ID, w.passes.retired, garden().ID)
	}
}

func TestAVaultThatIsNotTheOneItsFolderCarriesIsNotShown(t *testing.T) {
	mismatched := errors.New("the folder holds another vault")
	w := newTestWindow(t, func(deps *Deps) { deps.Vault.Identity = identity{mismatched: mismatched} })
	w.open(t, garden())

	err := w.scenarios.Switching.Show.Execute(t.Context(), studio(), nil)

	if !errors.Is(err, mismatched) {
		t.Fatalf("the switch returned %v, want %v", err, mismatched)
	}
	if w.pages.flushes != 0 {
		t.Errorf("the pages were flushed %d times for a vault that was mismatched", w.pages.flushes)
	}
}

func TestAWindowThatClosedShowsNoOtherVault(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())
	if !w.scenarios.Closing.Flush.Execute(t.Context()) {
		t.Fatal("the pages did not flush")
	}

	err := w.scenarios.Switching.Show.Execute(t.Context(), studio(), nil)

	if !errors.Is(err, window.ErrClosing) {
		t.Errorf("the switch returned %v, want %v", err, window.ErrClosing)
	}
}

func TestSwitchingStartsTheTasksOfTheNewVaultFromNothing(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())
	w.tasks.AddTask(work.Task{ID: work.MakingVectors, Title: "Indexing"})

	if err := w.scenarios.Switching.Show.Execute(t.Context(), studio(), nil); err != nil {
		t.Fatal(err)
	}

	if w.tasks.has(work.MakingVectors) {
		t.Error("the vectors of the first vault are still listed as being made")
	}
	if !w.tasks.has(work.WalkingNotes) {
		t.Error("the walk of the second vault is not listed")
	}
}
