package refresh_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/domain/resultcache"
)

func TestIndexChangeWithRefreshAtOnceIncludesTheNewNote(t *testing.T) {
	w := newWorld(t, 0)
	w.runDueSearches(t)
	if got := getResultPaths(w.readRecord(t, stormsID)); slices.Contains(got, "Harbor/Winter.md") {
		t.Fatalf("results = %v before the note is added", got)
	}

	w.addWinterNote()
	changedAt := w.clock.Advance(30 * time.Second)
	w.changeIndex(t)

	record := w.readRecord(t, stormsID)
	if got := getResultPaths(record); !slices.Contains(got, "Harbor/Winter.md") {
		t.Errorf("results = %v, want the new note included", got)
	}
	if !record.ComputedAt.Equal(changedAt) || record.StaleSince != nil {
		t.Errorf("computed at %v, stale since %s, want computed at %v and not stale", record.ComputedAt, formatMark(record.StaleSince), changedAt)
	}
}

func TestIndexChangeWithTenMinutesRunsAtTheEndOfTheCooldownNotBefore(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)
	first := w.readRecord(t, stormsID)
	searches := w.getStartCount()

	w.addWinterNote()
	w.clock.Advance(time.Minute)
	w.changeIndex(t)
	w.clock.Set(getStartTime().Add(10*time.Minute - time.Second))
	w.runDueSearches(t)

	if got := w.getStartCount(); got != searches {
		t.Errorf("%d searches before the cooldown ended, want none", got-searches)
	}
	if record := w.readRecord(t, stormsID); !record.ComputedAt.Equal(first.ComputedAt) || slices.Contains(getResultPaths(record), "Harbor/Winter.md") {
		t.Errorf("record = %+v before the cooldown ended, want the first results", record)
	}

	w.clock.Set(getStartTime().Add(10 * time.Minute))
	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if !slices.Contains(getResultPaths(record), "Harbor/Winter.md") || !record.ComputedAt.Equal(getStartTime().Add(10*time.Minute)) {
		t.Errorf("record = %+v at the end of the cooldown, want the new note included", record)
	}
	if record.StaleSince != nil {
		t.Errorf("stale since %s after the run, want not stale", formatMark(record.StaleSince))
	}
}

func TestIndexChangeAfterTheCooldownRunsAtOnce(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)

	w.addWinterNote()
	changedAt := w.clock.Advance(25 * time.Minute)
	w.changeIndex(t)

	record := w.readRecord(t, stormsID)
	if !slices.Contains(getResultPaths(record), "Harbor/Winter.md") || !record.ComputedAt.Equal(changedAt) {
		t.Errorf("record = %+v, want the new note included at once", record)
	}
}

func TestRunDueWithoutAnIndexChangeRunsNothing(t *testing.T) {
	w := newWorld(t, 0)
	w.runDueSearches(t)
	searches := w.getStartCount()
	if searches == 0 {
		t.Fatal("the first run did not search")
	}

	w.clock.Advance(time.Hour)
	w.runDueSearches(t)

	if got := w.getStartCount(); got != searches {
		t.Errorf("%d searches without a change, want none", got-searches)
	}
}

func TestRunDueRunsASearchWithoutACache(t *testing.T) {
	w := newWorld(t, 10)

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if got := getResultPaths(record); len(got) != 2 || !record.ComputedAt.Equal(getStartTime()) {
		t.Errorf("record = %+v, want the two matching passages computed now", record)
	}
}

func TestABrokenCacheFileRunsAgain(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)
	w.vault.Store().WriteFile(stormsID+".json", []byte(`{"path": "Searches/Winter storms.md", "results": [`))

	ranAt := w.clock.Advance(time.Minute)
	w.runDueSearches(t)

	if record := w.readRecord(t, stormsID); !record.ComputedAt.Equal(ranAt) || len(record.Results) != 2 {
		t.Errorf("record = %+v, want it computed again", record)
	}
}

func TestRecordKeepsTheFloorOrderRerankerAndIndexGenerationOfTheCore(t *testing.T) {
	w := newWorld(t, 10)

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if record.Path != stormsPath || record.QueryHash != resultcache.HashText(stormsQuery) {
		t.Errorf("record = %+v, want the path and the hash of the query", record)
	}
	if record.Floor != 0.4 || record.Order != resultcache.OrderReranked || record.Reranker != "Harbor reranker" || record.IndexGeneration != "gen-7" {
		t.Errorf("record = %+v, want the floor, the order, the reranker and the index generation of the core", record)
	}
	if got := getResultTexts(record); !slices.Equal(got, []string{galeText, mooringText}) {
		t.Errorf("results = %v, want them in the order of the core", got)
	}
	if record.Results[0].Score == nil || *record.Results[0].Score != 0.9 || record.Results[0].Title != "Gale" || record.Results[0].Kind != "note" {
		t.Errorf("result = %+v, want the score, the title and the kind of the core", record.Results[0])
	}
	if record.StaleSince != nil || record.FailedAt != nil || record.Failure != "" || len(record.Errors) != 0 {
		t.Errorf("record = %+v, want no mark, no failure and no problem", record)
	}
}

func TestRecordKeepsThePlaceAndAddressOfAResult(t *testing.T) {
	w := newWorld(t, 0)
	w.setStorms(getLogbookPage())

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if len(record.Results) != 1 {
		t.Fatalf("results = %+v, want the page of the logbook", record.Results)
	}
	got := record.Results[0]
	if got.Path != "Logbooks/1911.pdf" || got.Title != "1911" || got.Kind != "pdf" || got.Place == nil || got.Place.Page != 14 {
		t.Errorf("result = %+v, want the logbook, its title, its kind and page 14", got)
	}
	if got.Start != 48210 || got.Length != 2410 || got.Line != 380 || got.URI != "numen:Logbooks%2F1911.pdf?start=48210&length=2410" {
		t.Errorf("result = %+v, want its range, its line and its address", got)
	}
}

func TestRecordKeepsAWholeFileAsAFileResult(t *testing.T) {
	w := newWorld(t, 0)
	w.setStorms(host.SearchResult{Path: "Harbor/Pier plan.png", Title: "Pier plan", Kind: "image", IsWholeFile: true})

	w.runDueSearches(t)

	got := w.readRecord(t, stormsID).Results
	if len(got) != 1 || !got[0].IsWholeFile || got[0].Place != nil || got[0].URI != "numen:Harbor%2FPier%20plan.png" || got[0].Kind != "image" {
		t.Errorf("results = %+v, want the whole file with the address of the file", got)
	}
}

func TestRunReadsEveryPageOfTheResults(t *testing.T) {
	w := newWorld(t, 0)
	var results []host.SearchResult
	for i := range 250 {
		results = append(results, passage(fmt.Sprintf("Logbooks/Day %03d.md", i), mooringText, 0.5))
	}
	w.setStorms(results...)

	w.runDueSearches(t)

	if got := getResultPaths(w.readRecord(t, stormsID)); len(got) != 250 || got[249] != "Logbooks/Day 249.md" {
		t.Errorf("%d results, want all 250 in order", len(got))
	}
}

func TestRunDueStartsBackgroundSearchesAndRunNowInteractiveOnes(t *testing.T) {
	w := newWorld(t, 0)

	w.runDueSearches(t)
	w.runNow(t, stormsPath)

	started := w.searches.ListStarted()
	if len(started) != 2 {
		t.Fatalf("started = %+v, want two searches", started)
	}
	if !started[0].IsBackground || started[0].Priority != host.SearchPriorityBackground || started[0].Query != stormsQuery {
		t.Errorf("start of RunDue = %+v, want the query in the background, shown as no task of the core", started[0])
	}
	if started[1].IsBackground || started[1].Priority != host.SearchPriorityInteractive {
		t.Errorf("start of RunNow = %+v, want an interactive search the core shows", started[1])
	}
	if removed := w.searches.ListRemoved(); !slices.Equal(removed, []string{"search-1", "search-2"}) {
		t.Errorf("removed = %v, want both searches removed", removed)
	}
}

func TestCanceledRunRemovesItsSearch(t *testing.T) {
	w := newWorld(t, 0)
	release := w.searches.HoldSearch(stormsQuery)
	defer release()
	ctx, cancel := context.WithCancel(t.Context())
	w.host.setHook(func() { time.AfterFunc(50*time.Millisecond, cancel) })

	_, err := w.runner.RunNow(ctx, testVault, stormsPath)

	if err == nil {
		t.Fatal("RunNow = nil, want the end of the context")
	}
	if removed := w.searches.ListRemoved(); !slices.Equal(removed, []string{"search-1"}) {
		t.Errorf("removed = %v, want the canceled search removed", removed)
	}
}

func TestASearchWithoutAnIDIsCachedByAHashOfItsPath(t *testing.T) {
	w := newWorld(t, 0)
	note := formatSearchNote("", stormsQuery)
	w.vault.WriteFile(galesPath, note)

	w.runDueSearches(t)

	if name := resultcache.GetFileName("", galesPath); !w.hasRecord(t, name) {
		t.Errorf("no cache %s for the note without id", name)
	}
	if content, _ := w.vault.ReadContent(galesPath); content != note {
		t.Errorf("note = %q, want it unchanged", content)
	}
}

func TestRunOfANoteWithoutAQueryBlockRecordsTheProblem(t *testing.T) {
	w := newWorld(t, 0)
	w.vault.WriteFile(stormsPath, "---\nid: "+stormsID+"\ntype: search\n---\nStories of storms.\n")

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if len(record.Errors) != 1 || record.Errors[0].Message != "the note has no numen-query block" {
		t.Fatalf("errors = %+v, want the missing block", record.Errors)
	}
	if record.Errors[0].Line != 1 || record.Errors[0].Column != 1 || len(record.Results) != 0 || !isAt(record.FailedAt, getStartTime()) {
		t.Errorf("record = %+v, want the problem at line 1, column 1 of the query, no result, failed now", record)
	}
	if got := w.getStartCount(); got != 0 {
		t.Errorf("%d searches for a note with problems, want none", got)
	}
}

func TestRunRecordsTheProblemsOfTheCoreAtTheirLineInTheQuery(t *testing.T) {
	w := newWorld(t, 0)
	w.vault.WriteFile(stormsPath, formatSearchNote(stormsID, brokenQuery))

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if len(record.Errors) != 1 || record.Errors[0].Line != 1 || record.Errors[0].Column != 20 || record.Errors[0].Message == "" {
		t.Errorf("errors = %+v, want a problem at line 1, column 20 of the query", record.Errors)
	}
	if len(record.Results) != 0 {
		t.Errorf("results = %+v, want none", record.Results)
	}
}

func TestRunOfAProblemKeepsTheLastGoodResults(t *testing.T) {
	w := newWorld(t, 10)
	w.runDueSearches(t)

	w.vault.WriteFile(stormsPath, formatSearchNote(stormsID, brokenQuery))
	failedAt := w.clock.Advance(time.Minute)
	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if !slices.Equal(getResultTexts(record), []string{galeText, mooringText}) || !record.ComputedAt.Equal(getStartTime()) {
		t.Errorf("record = %+v, want the last good results and their time", record)
	}
	if len(record.Errors) != 1 || !isAt(record.FailedAt, failedAt) || record.Failure == "" || record.StaleSince == nil {
		t.Errorf("record = %+v, want the problem, the time of the failure and the search stale", record)
	}
}

func TestRunWithoutRerankingWritesFusedResults(t *testing.T) {
	w := newWorld(t, 0)
	w.searches.SetAnswer(stormsQuery, host.SearchState{Results: []host.SearchResult{passage("Logbooks/1911.md", mooringText, 0.7)}, ScoreKind: "meaning"})

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if record.Order != resultcache.OrderFused || record.Reranker != "" || record.FailedAt != nil || len(record.Warnings) != 0 {
		t.Errorf("record = %+v, want fused order, no reranker, no failure and no warning", record)
	}
}
