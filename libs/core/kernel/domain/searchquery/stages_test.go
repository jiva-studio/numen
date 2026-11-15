package searchquery_test

import (
	"slices"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func newRankedChunks(path string, score float64, starts ...int) searchquery.SearchResult {
	r := searchquery.SearchResult{Document: getNote(path), Score: new(score), Start: starts[0]}
	for _, start := range starts {
		r.Chunks = append(r.Chunks, searchquery.ChunkRange{Start: start, Length: 10})
	}
	return r
}

func TestSelectRerankUnitsReturnsTheChunksOfTheWindowInResultOrderUpToTheMost(t *testing.T) {
	r := searchquery.SearchResults{Results: []searchquery.SearchResult{
		newRankedChunks("a.md", 0.9, 40, 20),
		newRankedChunks("b.md", 0.8, 0),
		newRankedChunks("a.md", 0.7, 100),
		{Document: getNote("c.md")},
	}}

	units := searchquery.SelectRerankUnits(r, 2, 3)

	want := []searchquery.RerankUnit{{ResultIndex: 0, ChunkIndex: 1}, {ResultIndex: 0, ChunkIndex: 0}, {ResultIndex: 1, ChunkIndex: 0}}
	if !slices.Equal(units, want) {
		t.Errorf("units %+v, want %+v", units, want)
	}
	units = searchquery.SelectRerankUnits(r, 3, 10)
	if len(units) != 5 || units[4] != (searchquery.RerankUnit{ResultIndex: 3, ChunkIndex: -1}) {
		t.Errorf("units %+v, want the whole file of c.md last", units)
	}
}

func TestApplyRerankKeepsTheWindowScoresAMergedResultByItsBestChunkAndSorts(t *testing.T) {
	r := searchquery.SearchResults{ScoreKind: searchquery.ScoreMeaning, Results: []searchquery.SearchResult{
		newRankedChunks("a.md", 0.9, 0, 20),
		newRankedChunks("b.md", 0.8, 0),
		newRankedChunks("a.md", 0.7, 100),
		newRankedChunks("c.md", 0.6, 0),
	}}
	units := []searchquery.RerankUnit{{ResultIndex: 0, ChunkIndex: 0}, {ResultIndex: 0, ChunkIndex: 1}, {ResultIndex: 1, ChunkIndex: 0}}

	got := searchquery.ApplyRerank(r, 2, units, []float64{0.1, 0.5, 0.5})

	if got.ScoreKind != searchquery.ScoreRerank || len(got.Results) != 2 {
		t.Fatalf("results %+v, want the two results that had a chunk sent", got.Results)
	}
	first, second := got.Results[0], got.Results[1]
	if first.Document.Path != "a.md" || second.Document.Path != "b.md" {
		t.Errorf("order %s, %s, want a.md then b.md by path on equal scores", first.Document.Path, second.Document.Path)
	}
	if *first.Score != 0.5 || *first.RerankScore != 0.5 {
		t.Errorf("the merged result scores %v, want its best chunk 0.5", *first.Score)
	}
}

func TestApplySortByModifiedOrdersTheNewestFirstAndTitleDescReversesTitles(t *testing.T) {
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	older, newer := getNote("old.md"), getNote("new.md")
	older.ModifiedAt, older.Title = day, "Alpha"
	newer.ModifiedAt, newer.Title = day.Add(time.Hour), "Beta"
	tie := getNote("tie.md")
	tie.ModifiedAt, tie.Title = day, "Gamma"
	r := searchquery.SearchResults{Results: []searchquery.SearchResult{
		{Document: older, Score: new(0.2)}, {Document: tie, Score: new(0.9)}, {Document: newer, Score: new(0.1)},
	}}

	got := searchquery.ApplySort(r, searchquery.SortStage{Key: searchquery.SortModified, IsDescending: true})
	if paths := getResultPaths(got); !slices.Equal(paths, []string{"new.md", "tie.md", "old.md"}) {
		t.Errorf("by modified %v, want the newest first, then the tie by score", paths)
	}
	got = searchquery.ApplySort(r, searchquery.SortStage{Key: searchquery.SortTitle, IsDescending: true})
	if paths := getResultPaths(got); !slices.Equal(paths, []string{"tie.md", "new.md", "old.md"}) {
		t.Errorf("by title desc %v, want Gamma, Beta, Alpha", paths)
	}
}

func TestTruncateTextNeverSplitsARune(t *testing.T) {
	text := "ёёё"
	for most := range len(text) + 1 {
		cut := searchquery.TruncateText(text, most)
		if !utf8.ValidString(cut) || len(cut) > most {
			t.Errorf("TruncateText(%d) = %q", most, cut)
		}
	}
	if searchquery.TruncateText("abc", 10) != "abc" {
		t.Error("a short text was cut")
	}
}

func getResultPaths(r searchquery.SearchResults) []string {
	out := make([]string, len(r.Results))
	for i, one := range r.Results {
		out[i] = one.Document.Path
	}
	return out
}
