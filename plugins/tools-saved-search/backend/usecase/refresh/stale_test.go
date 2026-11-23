package refresh_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

func TestMarkStaleKeepsTheFirstStaleTime(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)

	firstMark := w.clock.Advance(time.Minute)
	if err := w.runner.MarkStale(t.Context(), testVault); err != nil {
		t.Fatal(err)
	}
	w.clock.Advance(4 * time.Minute)
	if err := w.runner.MarkStale(t.Context(), testVault); err != nil {
		t.Fatal(err)
	}

	if record := w.readRecord(t, stormsID); !isAt(record.StaleSince, firstMark) {
		t.Errorf("stale since %s, want the first mark %v", formatMark(record.StaleSince), firstMark)
	}
}

func TestMarkStaleCreatesNoCache(t *testing.T) {
	w := newWorld(t, 10)

	if err := w.runner.MarkStale(t.Context(), testVault); err != nil {
		t.Fatal(err)
	}

	if files, err := w.vault.Store().List(t.Context()); err != nil || len(files) != 0 {
		t.Errorf("store = %v, %v, want no file", files, err)
	}
}

func TestRunMarksTheRecordStaleWhenTheCoreReportsAStaleSearch(t *testing.T) {
	w := newWorld(t, 10)
	answer := getStormsAnswer(passage("Harbor/Gale.md", galeText, 0.9))
	answer.IsStale = true
	w.searches.SetAnswer(stormsQuery, answer)
	ranAt := w.clock.Advance(time.Minute)

	w.runNow(t, stormsPath)

	if record := w.readRecord(t, stormsID); !isAt(record.StaleSince, ranAt) || len(record.Results) != 1 {
		t.Errorf("record = %+v, want the results of the run, stale since the run", record)
	}
}

func TestStaleMarkPersistsAcrossRunners(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)
	w.addWinterNote()
	w.clock.Advance(time.Minute)
	if err := w.runner.MarkStale(t.Context(), testVault); err != nil {
		t.Fatal(err)
	}

	w.runner = w.newRunner()
	w.clock.Set(getStartTime().Add(10 * time.Minute))
	w.runDueSearches(t)

	if got := getResultPaths(w.readRecord(t, stormsID)); !slices.Contains(got, "Harbor/Winter.md") {
		t.Errorf("results = %v, want the new note included by the new runner", got)
	}
}

func TestEditOfTheNoteWithinTheCooldownRunsAtTheNextIndexChange(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)

	w.vault.WriteFile(stormsPath, formatSearchNote(stormsID, galesQuery))
	ranAt := w.clock.Advance(8 * time.Second)
	w.changeIndex(t)

	if record := w.readRecord(t, stormsID); !record.ComputedAt.Equal(ranAt) || len(record.Results) != 1 || record.StaleSince != nil {
		t.Errorf("record = %+v, want one result computed at once and not stale", record)
	}
}

func TestFailingProviderKeepsTheLastGoodResultsAndRetriesAfterTheCooldown(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)
	good := w.readRecord(t, stormsID)

	w.searches.SetAnswer(stormsQuery, host.SearchState{Errors: []host.SearchProblem{{Message: "the reranking service did not answer"}}})
	failedAt := w.clock.Advance(20 * time.Minute)
	w.changeIndex(t)

	failed := w.readRecord(t, stormsID)
	if !slices.Equal(getResultTexts(failed), getResultTexts(good)) || !failed.ComputedAt.Equal(good.ComputedAt) || failed.Order != good.Order {
		t.Errorf("record = %+v, want the last good results, order and time", failed)
	}
	if !isAt(failed.FailedAt, failedAt) || len(failed.Errors) != 1 || !strings.Contains(failed.Errors[0].Message, "reranking") || failed.StaleSince == nil {
		t.Errorf("failed at %s with %+v, stale since %s, want the problem now and the search still stale",
			formatMark(failed.FailedAt), failed.Errors, formatMark(failed.StaleSince))
	}
	if failed.Failure != "the reranking service did not answer" {
		t.Errorf("failure = %q, want the error of the run", failed.Failure)
	}

	w.changeMooringChunk()
	starts := w.getStartCount()
	w.clock.Advance(time.Minute)
	w.changeIndex(t)
	if got := w.getStartCount(); got != starts {
		t.Errorf("%d searches within the cooldown of the failed run, want none", got-starts)
	}

	retriedAt := w.clock.Advance(9 * time.Minute)
	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if !record.ComputedAt.Equal(retriedAt) || record.FailedAt != nil || record.Failure != "" || record.StaleSince != nil || len(record.Errors) != 0 {
		t.Errorf("record = %+v, want a good run that clears the failure", record)
	}
	if !slices.Contains(getResultTexts(record), mooringsText) {
		t.Errorf("results = %v, want the changed passage", getResultTexts(record))
	}
}

func TestHostFailureKeepsTheLastGoodResults(t *testing.T) {
	w := newWorld(t, 0)
	w.runDueSearches(t)
	good := w.readRecord(t, stormsID)

	w.host.setFailure(errors.New("the search of the host stopped"))
	failedAt := w.clock.Advance(time.Minute)
	if err := w.runner.MarkStale(t.Context(), testVault); err != nil {
		t.Fatal(err)
	}
	if err := w.runner.RunDue(t.Context(), testVault); err != nil {
		t.Logf("RunDue = %v", err)
	}

	record := w.readRecord(t, stormsID)
	if !slices.Equal(getResultTexts(record), getResultTexts(good)) || !record.ComputedAt.Equal(good.ComputedAt) {
		t.Errorf("record = %+v, want the last good results and their time", record)
	}
	if !isAt(record.FailedAt, failedAt) || !strings.Contains(record.Failure, "the search of the host stopped") {
		t.Errorf("failed at %s with %q, want the failure of the host now", formatMark(record.FailedAt), record.Failure)
	}
}
