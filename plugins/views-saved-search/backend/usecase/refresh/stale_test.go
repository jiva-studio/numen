package refresh_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

func TestMarkStaleKeepsTheFirstStaleTime(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)

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

func TestIndexChangeDuringARunMarksTheSearchStaleAgain(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)
	w.clock.Advance(time.Minute)
	var markedAt time.Time
	w.host.setHook(func() {
		w.host.setHook(nil)
		markedAt = w.clock.Advance(time.Second)
		marked := make(chan error, 1)
		go func() { marked <- w.runner.MarkStale(t.Context(), testVault) }()
		select {
		case err := <-marked:
			if err != nil {
				t.Errorf("MarkStale during the run = %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("MarkStale waited for the run to end")
		}
	})

	w.runNow(t, stormsPath)

	if record := w.readRecord(t, stormsID); !isAt(record.StaleSince, markedAt) {
		t.Errorf("stale since %s, want the mark of the index change during the run, %v", formatMark(record.StaleSince), markedAt)
	}
}

func TestStaleMarkPersistsAcrossRunners(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)
	w.addWinterNote()
	w.clock.Advance(time.Minute)
	if err := w.runner.MarkStale(t.Context(), testVault); err != nil {
		t.Fatal(err)
	}

	w.runner = w.newRunner()
	w.clock.Set(getStartTime().Add(10 * time.Minute))
	w.runDue(t)

	if got := getResultPaths(w.readRecord(t, stormsID)); !slices.Contains(got, "Harbor/Winter.md") {
		t.Errorf("results = %v, want the new note included by the new runner", got)
	}
}

func TestEditOfTheNoteWithinTheCooldownRunsAtTheNextIndexChange(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)

	w.vault.File(stormsPath, formatSearchNote(stormsID, stormsQuery, stormsRerank, 1))
	ranAt := w.clock.Advance(8 * time.Second)
	w.changeIndex(t)

	if record := w.readRecord(t, stormsID); !record.ComputedAt.Equal(ranAt) || len(record.Results) != 1 || record.StaleSince != nil {
		t.Errorf("record = %+v, want one result computed at once and not stale", record)
	}
}

func TestRerankerFailureKeepsTheLastGoodResultsAndRetriesAfterTheCooldown(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)
	good := w.readRecord(t, stormsID)

	w.standIn.Fail(connection.NewTryNextError(connection.ReasonUnavailable, 0))
	w.changeMooringChunk()
	failedAt := w.clock.Advance(20 * time.Minute)
	w.changeIndex(t)

	failed := w.readRecord(t, stormsID)
	if !slices.Equal(getResultTexts(failed), getResultTexts(good)) || !failed.ComputedAt.Equal(good.ComputedAt) || failed.Order != good.Order {
		t.Errorf("record = %+v, want the last good results, order and time", failed)
	}
	if !isAt(failed.FailedAt, failedAt) || !strings.Contains(failed.Failure, "reranking") || failed.StaleSince == nil {
		t.Errorf("failed at %s with %q, stale since %s, want the failure now and the search still stale",
			formatMark(failed.FailedAt), failed.Failure, formatMark(failed.StaleSince))
	}

	w.standIn.Fail(nil)
	w.standIn.ClearRequests()
	w.clock.Advance(time.Minute)
	w.changeIndex(t)
	if requests := w.standIn.GetRequests(); len(requests) != 0 {
		t.Errorf("rerank requests = %v within the cooldown of the failed run, want none", getDocuments(requests))
	}

	retriedAt := w.clock.Advance(9 * time.Minute)
	w.runDue(t)

	record := w.readRecord(t, stormsID)
	if !record.ComputedAt.Equal(retriedAt) || record.FailedAt != nil || record.Failure != "" || record.StaleSince != nil {
		t.Errorf("record = %+v, want a good run that clears the failure", record)
	}
	if !slices.Contains(getResultTexts(record), mooringsText) {
		t.Errorf("results = %v, want the changed passage", getResultTexts(record))
	}
}

func TestHostFailureKeepsTheLastGoodResults(t *testing.T) {
	w := newWorld(t, 0)
	w.runDue(t)
	good := w.readRecord(t, stormsID)

	w.search.FailEmbedding(errors.New("the embedding model stopped"))
	failedAt := w.clock.Advance(time.Minute)
	if err := w.runner.MarkStale(t.Context(), testVault); err != nil {
		t.Fatal(err)
	}
	_ = w.runner.RunDue(t.Context(), testVault)

	record := w.readRecord(t, stormsID)
	if !slices.Equal(getResultTexts(record), getResultTexts(good)) || !record.ComputedAt.Equal(good.ComputedAt) {
		t.Errorf("record = %+v, want the last good results and their time", record)
	}
	if !isAt(record.FailedAt, failedAt) || !strings.Contains(record.Failure, "the embedding model stopped") {
		t.Errorf("failed at %s with %q, want the failure of the host now", formatMark(record.FailedAt), record.Failure)
	}
}
