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
	go func() { _, _ = w.runner.CheckNoteQuery(t.Context(), testVault, stormsPath, stormsQuery) }()

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

func TestAnEditOfAnotherNoteKeepsTheCheckRunning(t *testing.T) {
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

func (w *world) awaitNotRunning(t *testing.T) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for w.runner.IsRunning(testVault, stormsPath) {
		select {
		case <-deadline:
			t.Fatal("the note still runs 5 seconds after its edit ended")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestAnEditIsRunningFromItsCheckToTheEndOfItsRun(t *testing.T) {
	w := newWorld(t, 0)
	changes, stop := w.runner.Watch(testVault, stormsPath)
	defer stop()
	release := w.searches.HoldSearch(stormsQuery)
	started := w.signalStarts()
	checked := make(chan error, 1)

	go func() {
		_, err := w.runner.CheckNoteQuery(t.Context(), testVault, stormsPath, stormsQuery)
		checked <- err
	}()
	awaitStart(t, started)
	select {
	case first := <-changes:
		if !first.IsRunning {
			t.Errorf("the watch received %+v when the check started, want a running change", first)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the watch received nothing when the check started")
	}
	if !w.runner.IsRunning(testVault, stormsPath) {
		t.Error("the note is not running while its edit is checked")
	}
	w.runner.StartRunNow(t.Context(), testVault, stormsPath)
	if !w.runner.IsRunning(testVault, stormsPath) {
		t.Error("the note is not running once its edit starts the run")
	}
	release()
	if err := <-checked; err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("the check answered %v", err)
	}
	awaitStart(t, started)

	w.awaitNotRunning(t)
	if w.readRecord(t, stormsID).ComputedAt.IsZero() {
		t.Error("the run of the edit wrote no results")
	}
}

func TestARunOfAnOlderEditKeepsTheCheckOfANewerEdit(t *testing.T) {
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
	awaitStart(t, started)
	release()

	if err := <-checked; err != nil {
		t.Errorf("the check of the newer edit answered %v, want none", err)
	}
}

func TestASecondStopOfAnEndedWatchKeepsALaterRun(t *testing.T) {
	w := newWorld(t, 0)
	release := w.searches.HoldSearch(stormsQuery)
	defer release()
	started := w.signalStarts()
	_, stop := w.runner.Watch(testVault, stormsPath)
	stop()

	w.runner.StartRunNow(t.Context(), testVault, stormsPath)
	awaitStart(t, started)
	stop()

	if removed := w.searches.ListRemoved(); len(removed) != 0 {
		t.Errorf("removed %v by a second stop of a watch that ended before the run, want none", removed)
	}
}
