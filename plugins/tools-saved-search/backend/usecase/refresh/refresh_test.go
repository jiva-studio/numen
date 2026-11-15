package refresh_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/resultcache"
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
	searches := w.host.getCallCount()

	w.addWinterNote()
	w.clock.Advance(time.Minute)
	w.changeIndex(t)
	w.clock.Set(getStartTime().Add(10*time.Minute - time.Second))
	w.runDueSearches(t)

	if got := w.host.getCallCount(); got != searches {
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
	searches := w.host.getCallCount()
	if searches == 0 {
		t.Fatal("the first run did not search")
	}

	w.clock.Advance(time.Hour)
	w.runDueSearches(t)

	if got := w.host.getCallCount(); got != searches {
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
	w.vault.Store().File(stormsID+".json", []byte(`{"path": "Searches/Winter storms.md", "results": [`))

	ranAt := w.clock.Advance(time.Minute)
	w.runDueSearches(t)

	if record := w.readRecord(t, stormsID); !record.ComputedAt.Equal(ranAt) || len(record.Results) != 2 {
		t.Errorf("record = %+v, want it computed again", record)
	}
}

func TestRecordStoresTheHashesAndTheRowOfTheRun(t *testing.T) {
	w := newWorld(t, 10)

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if record.Path != stormsPath || record.QueryHash != resultcache.HashText(stormsQuery) || record.RerankHash != resultcache.HashText(stormsRerank) {
		t.Errorf("record = %+v, want the path and the hashes of the query and the rerank text", record)
	}
	if record.Limit != 20 || record.Floor != 0.4 || record.CandidateCount == 0 || record.Order != api.OrderReranked {
		t.Errorf("record = %+v, want the default limit 20, the floor 0.4, the candidates counted, reranked", record)
	}
	if record.Reranker == nil || record.Reranker.ConnectionID != harborConnection || record.Reranker.Options["model"] != "harbor-rerank-a" {
		t.Errorf("reranker = %+v, want the row that answered", record.Reranker)
	}
	if got := getResultTexts(record); !slices.Equal(got, []string{galeText, mooringText}) {
		t.Errorf("results = %v, want them by the rerank scores", got)
	}
	if len(record.RerankScores) != 2 || record.RerankScores[resultcache.HashText(galeText)] != float64(float32(0.9)) {
		t.Errorf("rerank scores = %v, want one for each reranked text, by its hash", record.RerankScores)
	}
	if record.StaleSince != nil || record.FailedAt != nil || record.Failure != "" || len(record.Errors) != 0 {
		t.Errorf("record = %+v, want no mark, no failure and no problem", record)
	}
}

func TestRecordKeepsTheChunkIDsOfMergedResults(t *testing.T) {
	w := newWorld(t, 0)
	w.index(
		chunk("c1", "Harbor/Gale.md", 0, 400, galeText),
		chunk("c5", "Harbor/Gale.md", 400, 400, "Another storm at the pier."),
		chunk("c2", "Logbooks/1911.md", 0, 400, mooringText),
	)

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	i := slices.IndexFunc(record.Results, func(result resultcache.Result) bool { return result.Path == "Harbor/Gale.md" })
	if i < 0 {
		t.Fatalf("results = %v, want Harbor/Gale.md", getResultPaths(record))
	}
	if got := record.Results[i]; !slices.Equal(got.ChunkIDs, []string{"c1", "c5"}) || got.ChunkCount != 2 {
		t.Errorf("result = %+v, want the chunk ids c1 and c5 in start order", got)
	}
}

func TestRecordKeepsThePlaceAndAddressOfAResult(t *testing.T) {
	w := newWorld(t, 0)
	w.index(getLogbookPage())

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if len(record.Results) != 1 {
		t.Fatalf("results = %+v, want the page of the logbook", record.Results)
	}
	got := record.Results[0]
	if got.Path != "Logbooks/1911.pdf" || got.Title != "1911" || got.Place == nil || got.Place.Page != 14 {
		t.Errorf("result = %+v, want the logbook, its title and page 14", got)
	}
	if got.Start != 48210 || got.Length != 2410 || got.URI != "numen:Logbooks%2F1911.pdf?start=48210&length=2410" || !slices.Equal(got.ChunkIDs, []string{"ch-1911-14"}) {
		t.Errorf("result = %+v, want its range, its address and its chunk id", got)
	}
}

func TestASearchWithoutAnIDIsCachedByAHashOfItsPath(t *testing.T) {
	w := newWorld(t, 0)
	note := formatSearchNote("", stormsQuery, "", 0)
	w.vault.File(galesPath, note)

	w.runDueSearches(t)

	if name := resultcache.GetFileName("", galesPath); !w.hasRecord(t, name) {
		t.Errorf("no cache %s for the note without id", name)
	}
	if content, _ := w.vault.Content(galesPath); content != note {
		t.Errorf("note = %q, want it unchanged", content)
	}
}

func TestRunOfANoteWithoutAQueryBlockRecordsTheProblem(t *testing.T) {
	w := newWorld(t, 0)
	w.vault.File(stormsPath, "---\nid: "+stormsID+"\ntype: search\n---\nStories of storms.\n")

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if len(record.Errors) != 1 || record.Errors[0].Message != "the note has no numen-query block" {
		t.Fatalf("errors = %+v, want the missing block", record.Errors)
	}
	if record.Errors[0].Line != 5 || record.Errors[0].Column != 1 || len(record.Results) != 0 || !record.ComputedAt.Equal(getStartTime()) {
		t.Errorf("record = %+v, want the problem at line 5, column 1, no result, computed now", record)
	}
	if got := w.host.getCallCount(); got != 0 {
		t.Errorf("%d searches for a note with problems, want none", got)
	}
}

func TestRunRecordsCompileProblemsAtTheirLineInTheFile(t *testing.T) {
	w := newWorld(t, 0)
	// The block opens on line 5: the query is on line 6.
	w.vault.File(stormsPath, formatSearchNote(stormsID, `meaning("storm") &&`, "", 0))

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if len(record.Errors) == 0 || record.Errors[0].Line != 6 || record.Errors[0].Column != 20 || record.Errors[0].Message == "" {
		t.Errorf("errors = %+v, want a problem at line 6, column 20 of the file", record.Errors)
	}
	if len(record.Results) != 0 {
		t.Errorf("results = %+v, want none", record.Results)
	}
}

func TestRunWithoutAConnectionWritesFusedResults(t *testing.T) {
	w := newWorld(t, 0)
	w.rows.Clear()

	w.runDueSearches(t)

	record := w.readRecord(t, stormsID)
	if record.Order != api.OrderFused || record.Reranker != nil || len(record.RerankScores) != 0 {
		t.Errorf("record = %+v, want fused order, no reranker and no rerank score", record)
	}
	if got := getResultTexts(record); !slices.Equal(got, []string{mooringText, galeText}) || record.FailedAt != nil || len(record.Warnings) != 0 {
		t.Errorf("record = %+v, want the fused order without a failure or warning", record)
	}
}
