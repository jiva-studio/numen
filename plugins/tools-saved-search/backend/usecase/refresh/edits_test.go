package refresh_test

import (
	"context"
	"errors"
	"testing"
	"time"
)

func (w *world) signalStarts() <-chan struct{} {
	started := make(chan struct{}, 16)
	w.host.setHook(func() { started <- struct{}{} })
	return started
}

func awaitStart(t *testing.T, started <-chan struct{}) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("no search started within 5 seconds")
	}
}

func (w *world) awaitRemovedCount(t *testing.T, count int) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for len(w.searches.ListRemoved()) < count {
		select {
		case <-deadline:
			t.Fatalf("removed %v, want %d searches removed", w.searches.ListRemoved(), count)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestANewEditOfANoteRemovesTheSearchOfThePreviousEdit(t *testing.T) {
	w := newWorld(t, 0)
	release := w.searches.HoldSearch(stormsQuery)
	defer release()
	started := w.signalStarts()

	w.runner.StartRunNow(t.Context(), testVault, stormsPath)
	awaitStart(t, started)
	w.runner.StartRunNow(t.Context(), testVault, stormsPath)
	awaitStart(t, started)

	if removed := w.searches.ListRemoved(); len(removed) != 1 || removed[0] != "search-1" {
		t.Errorf("removed %v when the second edit started, want the search of the first", removed)
	}
	if active := w.host.getActiveCount(); active != 1 {
		t.Errorf("%d searches on their way, want the one of the second edit", active)
	}
}

func TestTheEndOfTheLastWatchOfANoteRemovesTheSearchOfItsEdit(t *testing.T) {
	w := newWorld(t, 0)
	release := w.searches.HoldSearch(stormsQuery)
	defer release()
	started := w.signalStarts()
	_, stopFirst := w.runner.Watch(testVault, stormsPath)
	_, stopSecond := w.runner.Watch(testVault, stormsPath)

	w.runner.StartRunNow(t.Context(), testVault, stormsPath)
	awaitStart(t, started)
	stopFirst()
	if removed := w.searches.ListRemoved(); len(removed) != 0 {
		t.Errorf("removed %v while a watch stays, want none", removed)
	}
	stopSecond()

	w.awaitRemovedCount(t, 1)
	if active := w.host.getActiveCount(); active != 0 {
		t.Errorf("%d searches on their way after the last watch ended, want none", active)
	}
}

func TestANewEditOfANoteCancelsTheCheckOfThePreviousEdit(t *testing.T) {
	w := newWorld(t, 0)
	release := w.searches.HoldSearch(stormsQuery)
	defer release()
	started := w.signalStarts()
	checked := make(chan error, 1)

	go func() {
		_, err := w.runner.CheckNoteQuery(t.Context(), testVault, stormsPath, stormsQuery)
		checked <- err
	}()
	awaitStart(t, started)
	w.runner.StartRunNow(t.Context(), testVault, stormsPath)

	select {
	case err := <-checked:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the check of the first edit answered %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the check of the first edit still runs")
	}
	w.awaitRemovedCount(t, 1)
}

func TestAnEditOfAnotherNoteLeavesTheCheckRunning(t *testing.T) {
	w := newWorld(t, 0)
	addGalesNote(w)
	release := w.searches.HoldSearch(stormsQuery)
	started := w.signalStarts()
	checked := make(chan error, 1)

	go func() {
		_, err := w.runner.CheckNoteQuery(t.Context(), testVault, stormsPath, stormsQuery)
		checked <- err
	}()
	awaitStart(t, started)
	w.runner.StartRunNow(t.Context(), testVault, galesPath)
	awaitStart(t, started)
	release()

	if err := <-checked; err != nil {
		t.Errorf("the check answered %v after an edit of another note, want none", err)
	}
}
