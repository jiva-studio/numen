package refresh_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

const (
	tidesPath   = "Searches/A tides.md"
	tidesQuery  = `words("tide")`
	quaysPath   = "Searches/B quays.md"
	quaysQuery  = `words("quay")`
	buoysPath   = "Searches/C buoys.md"
	buoysQuery  = `words("buoy")`
	wrecksPath  = "Searches/D wrecks.md"
	wrecksQuery = `words("wreck")`
)

func newBatchWorld(t *testing.T) *world {
	t.Helper()
	w := newWorld(t, 0)
	w.vault.Remove(stormsPath)
	for _, note := range []struct{ path, query, text string }{
		{tidesPath, tidesQuery, "The tide at the quay."},
		{quaysPath, quaysQuery, "The north quay."},
		{buoysPath, buoysQuery, "A red buoy."},
		{wrecksPath, wrecksQuery, "The wreck of the Maria."},
	} {
		w.searches.SetAnswer(note.query, host.SearchState{Results: []host.SearchResult{passage("Harbor/"+note.text+".md", note.text, 1)}, ScoreKind: "words"})
		if note.path != wrecksPath {
			w.vault.WriteFile(note.path, formatSearchNote("", note.query))
		}
	}
	return w
}

func startBatch(t *testing.T, w *world) <-chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- w.runner.RunDue(t.Context(), testVault) }()
	waitForStart(t, w, tidesQuery)
	return done
}

func waitForStart(t *testing.T, w *world, query string) {
	t.Helper()
	waiting, stopWaiting := context.WithTimeout(t.Context(), 5*time.Second)
	defer stopWaiting()
	for !slices.ContainsFunc(w.searches.ListStarted(), func(start host.SearchStart) bool { return start.Query == query }) {
		if waiting.Err() != nil {
			t.Fatalf("no search of %s started within 5 seconds", query)
		}
		time.Sleep(time.Millisecond)
	}
}

func listStartedQueries(w *world) []string {
	var queries []string
	for _, start := range w.searches.ListStarted() {
		queries = append(queries, start.Query)
	}
	return queries
}

func TestAnInteractiveRunGoesBeforeTheRestOfTheBatch(t *testing.T) {
	w := newBatchWorld(t)
	release := w.searches.HoldSearch(tidesQuery)
	defer release()
	done := startBatch(t, w)

	w.vault.WriteFile(wrecksPath, formatSearchNote("", wrecksQuery))
	record := w.runNow(t, wrecksPath)

	if len(record.Results) != 1 {
		t.Errorf("record = %+v, want the wrecks computed", record)
	}
	started := listStartedQueries(w)
	if slices.Contains(started, quaysQuery) || slices.Contains(started, buoysQuery) {
		t.Errorf("started = %v when the interactive run ended, want the rest of the batch after it", started)
	}
	release()
	if err := <-done; err != nil {
		t.Fatalf("RunDue = %v", err)
	}
	started = listStartedQueries(w)
	if i := slices.Index(started, wrecksQuery); i < 0 || i > slices.Index(started, quaysQuery) || i > slices.Index(started, buoysQuery) {
		t.Errorf("started = %v, want the wrecks before the quays and the buoys", started)
	}
	for _, notePath := range []string{tidesPath, quaysPath, buoysPath} {
		if state, err := w.runner.ReadState(t.Context(), testVault, notePath); err != nil || state.Record.ComputedAt.IsZero() {
			t.Errorf("state of %s = %+v, %v, want the batch to run it", notePath, state, err)
		}
	}
}

func TestANeverComputedSearchIsRunningWhileQueuedInTheBatch(t *testing.T) {
	w := newBatchWorld(t)
	release := w.searches.HoldSearch(tidesQuery)
	defer release()
	done := startBatch(t, w)

	state, err := w.runner.ReadState(t.Context(), testVault, buoysPath)
	if err != nil {
		t.Fatal(err)
	}
	if !state.IsRunning || state.HasRecord {
		t.Errorf("state = %+v while the buoys wait for their turn, want a search never computed and running", state)
	}

	release()
	if err := <-done; err != nil {
		t.Fatalf("RunDue = %v", err)
	}
	if w.runner.IsRunning(testVault, buoysPath) {
		t.Error("the buoys are running after the batch, want them done")
	}
}

func TestABatchThatEndsWhileWaitingForTheVaultUnqueuesItsSearches(t *testing.T) {
	w := newBatchWorld(t)
	w.vault.WriteFile(wrecksPath, formatSearchNote("", wrecksQuery))
	releaseTides := w.searches.HoldSearch(tidesQuery)
	defer releaseTides()
	releaseWrecks := w.searches.HoldSearch(wrecksQuery)
	defer releaseWrecks()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- w.runner.RunDue(ctx, testVault) }()
	waitForStart(t, w, tidesQuery)
	interactive := make(chan struct{})
	go func() {
		defer close(interactive)
		if _, err := w.runner.RunNow(t.Context(), testVault, wrecksPath); err != nil {
			t.Errorf("RunNow = %v", err)
		}
	}()
	waitForStart(t, w, wrecksQuery)

	cancel()
	<-done

	for _, notePath := range []string{tidesPath, quaysPath, buoysPath} {
		if w.runner.IsRunning(testVault, notePath) {
			t.Errorf("%s is running after the batch ended, want none queued", notePath)
		}
	}
	releaseWrecks()
	<-interactive
}
