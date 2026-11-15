package service_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/resultcache"
)

func addWinterNote(f *fixture) {
	f.vault.File("Harbor/Winter.md", "# Winter\n\n"+winterText+"\n")
	f.search.With(testVault).SetIndexedResults(getIndex(mooringText,
		host.SearchResult{ChunkID: "c5", Path: "Harbor/Winter.md", Text: winterText, Start: 0, Length: 400}))
}

func TestHandleEventWithRefreshAtOnceIncludesTheNewNote(t *testing.T) {
	f := newFixture(t, 0)
	f.addStormsNote()
	f.handleIndexChange(t)

	addWinterNote(f)
	f.clock.Advance(30 * time.Second)
	f.handleIndexChange(t)

	if got := getResultPaths(f.getResults(t, stormsPath)); !slices.Contains(got, "Harbor/Winter.md") {
		t.Errorf("results = %v, want the new note included", got)
	}
}

func TestHandleEventMarksTheSearchesOfTheVaultAndRunsTheDueOnes(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	f.handleIndexChange(t)
	if record := f.readRecord(t, stormsID); len(record.Results) != 2 {
		t.Fatalf("record = %+v, want the first run of the search without a cache", record)
	}

	addWinterNote(f)
	markedAt := f.clock.Advance(time.Minute)
	f.handleIndexChange(t)

	record := f.readRecord(t, stormsID)
	if record.StaleSince == nil || !record.StaleSince.Equal(markedAt) || slices.ContainsFunc(record.Results, func(r resultcache.Result) bool { return r.Path == "Harbor/Winter.md" }) {
		t.Errorf("record = %+v, want the search marked stale and not run within the cooldown", record)
	}

	f.clock.Set(getStartTime().Add(10 * time.Minute))
	f.handleIndexChange(t)

	if got := getResultPaths(f.getResults(t, stormsPath)); !slices.Contains(got, "Harbor/Winter.md") {
		t.Errorf("results = %v at the end of the cooldown, want the new note included", got)
	}
}

func TestHandleEventReportsTheRefreshAndFinishesIt(t *testing.T) {
	f := newFixture(t, 0)
	f.addStormsNote()

	f.handleIndexChange(t)

	if record := f.readRecord(t, stormsID); len(record.Results) != 2 {
		t.Errorf("record = %+v, want the run of the event", record)
	}

	if tasks := f.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks = %+v after the event, want the refresh finished", tasks)
	}
}

func TestHandleEventIgnoresAnotherEvent(t *testing.T) {
	f := newFixture(t, 0)
	f.addStormsNote()

	for _, event := range []string{"onStartup", "onSearchSourceReady:" + testVault, "onIndexChanged"} {
		if err := f.app.HandleEvent(t.Context(), event); err != nil {
			t.Errorf("HandleEvent(%q) = %v, want nil", event, err)
		}
	}

	if f.host.getCallCount() != 0 {
		t.Errorf("%d searches for other events, want none", f.host.getCallCount())
	}
	if files, err := f.vault.Store().List(t.Context()); err != nil || len(files) != 0 {
		t.Errorf("store = %v, %v, want nothing written", files, err)
	}
}
