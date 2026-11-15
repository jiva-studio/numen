package searchquery_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
)

func newDeepWorld(t *testing.T) world {
	t.Helper()
	w := newWorld(t, map[string]string{
		"deep/target.md": "The harbor at dusk, where the boats come home. " + strings.Repeat("calm water ", 10),
		"out/many.md":    repeatSentences("near", 500),
	})
	w.cutBySize(t, "deep/target.md", 40)
	w.cutBySize(t, "out/many.md", 40)
	w.vectorize(t, "deep/target.md", func(chunk.Record) []float32 { return newVector(dimensions / 4) })
	w.vectorize(t, "out/many.md", func(chunk.Record) []float32 { return newVector(0) })
	return w
}

func TestAMeaningInAScopeFindsItsChunkThoughNearerChunksAreOutsideIt(t *testing.T) {
	w := newDeepWorld(t)
	find := w.newFind(texts{"a": newVector(0)})
	find.Limits.CandidatesPerCondition = 5
	got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `meaning("a") && path.startsWith("deep/")`})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) == 0 || got.Results[0].Document.Path != "deep/target.md" {
		t.Fatalf("found %v, want the chunk under deep/", getPaths(got))
	}
	for _, r := range got.Results {
		if r.Document.Path != "deep/target.md" {
			t.Errorf("found %s outside the scope", r.Document.Path)
		}
	}
}

func TestAMeaningInAScopeOverTheExactScanFindsItsChunkByTheApproximateSearch(t *testing.T) {
	w := newDeepWorld(t)
	for _, text := range []string{`meaning("a") && path.startsWith("deep/")`, `meaning("a") && !path.startsWith("out/")`} {
		find := w.newFind(texts{"a": newVector(0)})
		find.Limits.ExactScanChunks = 0
		find.Limits.CandidatesPerCondition = 1
		find.Limits.CoarseOversample = 1
		got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: text})
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Results) != 1 || got.Results[0].Document.Path != "deep/target.md" {
			t.Errorf("%s found %v, want the chunk under deep/", text, getPaths(got))
		}
	}
}

func TestAMeaningWithFewChunksAboveTheFloorSearchesOnceWithoutAWarning(t *testing.T) {
	w := newWorld(t, map[string]string{
		"near.md": "The harbor at dusk, where the boats come home. " + strings.Repeat("calm water ", 6),
		"far.md":  repeatSentences("far", 200),
	})
	w.cutBySize(t, "near.md", 40)
	w.cutBySize(t, "far.md", 40)
	w.vectorize(t, "near.md", func(chunk.Record) []float32 { return newVector(0) })
	w.vectorize(t, "far.md", func(chunk.Record) []float32 { return newVector(dimensions / 2) })
	find, counted := w.newCountedFind(texts{"a": newVector(0)})
	find.Limits.ExactScanChunks = 0
	find.Limits.CandidatesPerCondition = 10
	find.Limits.CoarseOversample = 1
	find.Limits.MaxCoarseCandidates = 80
	got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `meaning("a")`})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) == 0 || got.Results[0].Document.Path != "near.md" {
		t.Errorf("found %v, want near.md", getPaths(got))
	}
	if len(got.Warnings) != 0 || counted.coarseSearches != 1 {
		t.Errorf("searched %d times with warnings %v, want once and no warning", counted.coarseSearches, got.Warnings)
	}
}

func TestAMeaningWhoseNearestChunksAreOutOfScopeWidensTheSearch(t *testing.T) {
	w := newDeepWorld(t)
	find, counted := w.newCountedFind(texts{"a": newVector(0)})
	find.Limits.ExactScanChunks = 0
	find.Limits.CandidatesPerCondition = 1
	find.Limits.CoarseOversample = 1
	got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `meaning("a") && !path.startsWith("out/")`})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 || got.Results[0].Document.Path != "deep/target.md" {
		t.Errorf("found %v, want the chunk under deep/", getPaths(got))
	}
	if counted.coarseSearches < 2 || len(got.Warnings) != 0 {
		t.Errorf("searched %d times with warnings %v, want a wider search and no warning", counted.coarseSearches, got.Warnings)
	}
}

func TestWordsOrMeaningFusesAndTheChunkMatchedByBothRanksFirst(t *testing.T) {
	w := newWorld(t, map[string]string{
		"a.md": "The tide comes over the sand at night.",
		"b.md": "A tide table, printed on paper.",
		"c.md": "Waves and salt water, far from land.",
	})
	for _, p := range []string{"a.md", "b.md", "c.md"} {
		w.cutBySize(t, p, 400)
	}
	w.vectorize(t, "a.md", func(chunk.Record) []float32 { return newVector(dimensions / 8) })
	w.vectorize(t, "b.md", func(chunk.Record) []float32 { return newVector(dimensions) })
	w.vectorize(t, "c.md", func(chunk.Record) []float32 { return newVector(0) })
	got, err := w.newFind(texts{"sea": newVector(0)}).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide") || meaning("sea")`})
	if err != nil {
		t.Fatal(err)
	}
	if got.ScoreKind != searchquerydomain.ScoreFused {
		t.Errorf("the score kind is %v, want fused", got.ScoreKind)
	}
	if len(got.Results) != 3 || got.Results[0].Document.Path != "a.md" {
		t.Errorf("found %v, want a.md first of three", getPaths(got))
	}
}

func TestAKindAloneReturnsOneWholeFilePerDocumentByPath(t *testing.T) {
	w := newWorld(t, map[string]string{"notes/x.md": "A note."})
	for _, p := range []string{"books/b.pdf", "books/a.pdf"} {
		if err := w.db.Chunks().SaveSource(t.Context(), w.v.ID, chunk.Source{Path: p, Kind: "file", Size: 4, MTime: 1, ContentHash: "h-" + p}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := w.newFind(nil).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `kind == "pdf"`})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(getPaths(got), ",") != "books/a.pdf,books/b.pdf" {
		t.Fatalf("found %v, want the two pdfs by path", getPaths(got))
	}
	for _, r := range got.Results {
		if !r.IsWholeFile() || r.Score != nil || r.Document.Kind != searchquerydomain.KindPDF || r.Document.Title == "" {
			t.Errorf("the result is %+v, want a whole pdf with a title and no score", r)
		}
	}
}

func TestAMeaningWithoutAModelReturnsErrNoModelAndReadsNothing(t *testing.T) {
	find := searchquery.NewFindSearchResults(searchquery.FindSearchResultsDeps{Parser: celquery.NewParser(searchquerydomain.DefaultLimits())})
	_, err := find.Execute(t.Context(), newWorld(t, nil).v, searchquerydomain.Request{Text: `meaning("a")`})
	if !errors.Is(err, searchquerydomain.ErrNoModel) {
		t.Errorf("returned %v, want ErrNoModel", err)
	}
}

func TestAWordsExpressionTheIndexRejectsIsAProblemAtItsPosition(t *testing.T) {
	w := newWorld(t, map[string]string{"a.md": "Some words here."})
	w.cutBySize(t, "a.md", 400)
	_, err := w.newFind(nil).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("here") || words("\"unterminated")`})
	parseError, ok := errors.AsType[*searchquerydomain.ParseError](err)
	if !ok || len(parseError.Problems) != 1 {
		t.Fatalf("returned %v, want one problem", err)
	}
	if got := parseError.Problems[0].Position; got != (searchquerydomain.Position{Line: 1, Column: 18}) {
		t.Errorf("the problem is at %+v, want 1:18", got)
	}
}

func TestTwoMatchingChunksCloseInOneFileAreOneResult(t *testing.T) {
	text := "Tide one. " + strings.Repeat("a", 190) + strings.Repeat("b", 100) + "Tide two. " + strings.Repeat("c", 190)
	w := newWorld(t, map[string]string{"a.md": text})
	w.cutRanges(t, "a.md", [2]int{0, 200}, [2]int{300, 500})
	got, err := w.newFind(nil).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide")`})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 || len(got.Results[0].Chunks) != 2 || got.Results[0].Start != 0 || got.Results[0].Length != 500 {
		t.Errorf("found %+v, want one result of two chunks", got.Results)
	}
}

func TestFiveChunksFarApartAreFiveResultsAmongTheOthersByScore(t *testing.T) {
	var text string
	for _, letter := range []string{"v", "w", "x", "y", "z"} {
		text += strings.Repeat(letter, 100) + strings.Repeat(" ", 400)
	}
	w := newWorld(t, map[string]string{"big.md": text, "small.md": "One passage alone in its file."})
	w.cutRanges(t, "big.md", [2]int{0, 100}, [2]int{500, 600}, [2]int{1000, 1100}, [2]int{1500, 1600}, [2]int{2000, 2100})
	w.cutBySize(t, "small.md", 400)
	w.vectorize(t, "big.md", func(r chunk.Record) []float32 { return newVector(r.Start / 500 * 64) })
	w.vectorize(t, "small.md", func(chunk.Record) []float32 { return newVector(32) })
	got, err := w.newFind(texts{"a": newVector(0)}).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `meaning("a")`})
	if err != nil {
		t.Fatal(err)
	}
	want := "big.md,small.md,big.md,big.md,big.md,big.md"
	if strings.Join(getPaths(got), ",") != want {
		t.Errorf("found %v, want %s", getPaths(got), want)
	}
	again, err := w.newFind(texts{"a": newVector(0)}).Execute(t.Context(), w.v, searchquerydomain.Request{Text: `meaning("a")`})
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Errorf("a second run returned other results: %v", err)
	}
}

func TestMoreCandidatesThanTheLimitAreTruncatedWithAWarning(t *testing.T) {
	text := strings.Repeat("tide "+strings.Repeat("a", 295), 3)
	w := newWorld(t, map[string]string{"a.md": text})
	w.cutRanges(t, "a.md", [2]int{0, 300}, [2]int{300, 600}, [2]int{600, 900})
	find := w.newFind(nil)
	find.Limits.MaxCandidates = 2
	got, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `words("tide")`})
	if err != nil {
		t.Fatal(err)
	}
	if got.Candidates != 2 || len(got.Warnings) != 1 {
		t.Errorf("evaluated %d candidates with warnings %v, want 2 and one warning", got.Candidates, got.Warnings)
	}
}

func TestDocumentConditionsOverTheDocumentLimitAreRefused(t *testing.T) {
	w := newWorld(t, map[string]string{"a.md": "A.", "b.md": "B."})
	find := w.newFind(nil)
	find.Limits.MaxDocuments = 1
	_, err := find.Execute(t.Context(), w.v, searchquerydomain.Request{Text: `kind == "note"`})
	if !errors.Is(err, searchquerydomain.ErrLimitExceeded) || !strings.Contains(err.Error(), "MaxDocuments") {
		t.Errorf("returned %v, want ErrLimitExceeded naming MaxDocuments", err)
	}
}

func TestACanceledQueryReturnsTheCancellation(t *testing.T) {
	w := newWorld(t, map[string]string{"a.md": "Tide."})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := w.newFind(nil).Execute(ctx, w.v, searchquerydomain.Request{Text: `words("tide")`})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("returned %v, want context.Canceled", err)
	}
}
