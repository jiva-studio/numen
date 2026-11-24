package refresh_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

func (w *world) watchStorms(t *testing.T) func() {
	t.Helper()
	_, stop := w.runner.Watch(testVault, stormsPath)
	t.Cleanup(stop)
	return stop
}

func TestIndexChangeOfAnOpenSearchRunsAfterTheDebounceNotTheCooldown(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)
	w.watchStorms(t)
	searches := w.getStartCount()

	w.addWinterNote()
	w.clock.Advance(time.Minute)
	w.changeIndex(t)
	if got := w.getStartCount(); got != searches {
		t.Errorf("%d searches within the debounce, want none", got-searches)
	}

	ranAt := w.clock.Advance(2 * time.Second)
	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if !slices.Contains(getResultPaths(record), "Harbor/Winter.md") || !record.ComputedAt.Equal(ranAt) || record.StaleSince != nil {
		t.Errorf("record = %+v, want the new note included at the end of the debounce and not stale", record)
	}
}

func TestIndexChangeOfAClosedSearchWaitsTheCooldown(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)
	stop := w.watchStorms(t)
	stop()
	searches := w.getStartCount()

	w.addWinterNote()
	w.clock.Advance(time.Minute)
	w.changeIndex(t)
	w.clock.Advance(time.Minute)
	w.runDueSearches(t)

	if got := w.getStartCount(); got != searches {
		t.Errorf("%d searches within the cooldown of a closed search, want none", got-searches)
	}
}

func TestClosingTheTabWithinTheDebounceFallsBackToTheCooldown(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)
	stop := w.watchStorms(t)
	searches := w.getStartCount()

	w.addWinterNote()
	w.clock.Advance(time.Minute)
	w.changeIndex(t)
	stop()
	w.clock.Advance(2 * time.Second)
	w.runDueSearches(t)

	if got := w.getStartCount(); got != searches {
		t.Errorf("%d searches after the tab closed within the debounce, want none before the cooldown ends", got-searches)
	}

	w.clock.Set(getStartTime().Add(10 * time.Minute))
	w.runDueSearches(t)

	if record := w.readRecord(t, stormsID); !slices.Contains(getResultPaths(record), "Harbor/Winter.md") {
		t.Errorf("results = %v at the end of the cooldown, want the new note included", getResultPaths(record))
	}
}

func TestEditRunsAtOnceRightAfterARun(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)

	w.vault.WriteFile(stormsPath, formatSearchNote(stormsID, galesQuery))
	editedAt := w.clock.Advance(time.Second)
	record := w.runNow(t, stormsPath)

	if !record.ComputedAt.Equal(editedAt) || len(record.Results) != 1 || record.StaleSince != nil {
		t.Errorf("record = %+v, want the one result of the edited query at once", record)
	}
}

func TestEditOfAnOpenSearchMarkedStaleByTheIndexRunsAgainAfterTheDebounce(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)
	w.watchStorms(t)

	w.vault.WriteFile(stormsPath, formatSearchNote(stormsID, galesQuery))
	answer := host.SearchState{Results: []host.SearchResult{passage("Harbor/Gale.md", galeText, 1)}, ScoreKind: "words", IsStale: true}
	w.searches.SetAnswer(galesQuery, answer)
	w.clock.Advance(time.Second)
	w.runNow(t, stormsPath)
	answer.IsStale = false
	w.searches.SetAnswer(galesQuery, answer)
	w.clock.Advance(time.Second)
	w.changeIndex(t)

	ranAt := w.clock.Advance(time.Second)
	w.runDueSearches(t)

	if record := w.readRecord(t, stormsID); !record.ComputedAt.Equal(ranAt) || record.StaleSince != nil {
		t.Errorf("record = %+v, want a run at the end of the debounce that is not stale", record)
	}
}
