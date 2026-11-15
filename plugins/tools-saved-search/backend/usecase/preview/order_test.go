package preview_test

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
)

func getRanges(out api.PreviewOutput) [][3]int {
	ranges := make([][3]int, len(out.Results))
	for i, result := range out.Results {
		ranges[i] = [3]int{result.Start, result.Length, result.ChunkCount}
	}
	return ranges
}

func TestPreviewReturnsTwoAdjacentChunksAsOneResult(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Logs/Storm Log.md", 0, 500, "The tide rose at six."),
		chunk("c2", "Logs/Storm Log.md", 500, 500, "The tide fell at noon."),
	)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	if got := getRanges(out); !slices.Equal(got, [][3]int{{0, 1000, 2}}) {
		t.Errorf("results (start, length, chunks) = %v, want one result over both chunks", got)
	}
}

func TestPreviewJoinsChunksWithinTheGap(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Logs/Storm Log.md", 0, 500, "The tide rose at six."),
		chunk("c2", "Logs/Storm Log.md", 650, 350, "The tide fell at noon."),
		chunk("c3", "Logs/Storm Log.md", 1200, 300, "The tide rose at dusk."),
	)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	if got := getRanges(out); !slices.Equal(got, [][3]int{{0, 1500, 3}}) {
		t.Errorf("results (start, length, chunks) = %v, want one result: gaps of 150 and 200 bytes join", got)
	}
}

func TestPreviewKeepsChunksBeyondTheGapApart(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Logs/Storm Log.md", 0, 500, "The tide rose at six."),
		chunk("c2", "Logs/Storm Log.md", 701, 300, "The tide fell at noon."),
	)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	if got := getRanges(out); !slices.Equal(got, [][3]int{{0, 500, 1}, {701, 300, 1}}) {
		t.Errorf("results (start, length, chunks) = %v, want two results 201 bytes apart", got)
	}
}

func TestPreviewMergesWithTheGapOfTheSettings(t *testing.T) {
	w := newWorld(t)
	w.config.MergeGapBytes = 50
	w.index(
		chunk("c1", "Logs/Storm Log.md", 0, 500, "The tide rose at six."),
		chunk("c2", "Logs/Storm Log.md", 650, 350, "The tide fell at noon."),
	)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	if got := getRanges(out); len(got) != 2 {
		t.Errorf("results (start, length, chunks) = %v, want two results under a gap of 50 bytes", got)
	}
}

func TestPreviewKeepsChunksOfTwoFilesApart(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Logs/A.md", 0, 500, "The tide rose at six."),
		chunk("c2", "Logs/B.md", 500, 500, "The tide fell at noon."),
	)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	if got := getPaths(out); len(got) != 2 {
		t.Errorf("results = %v, want one result for each file", got)
	}
}

func TestPreviewShowsTheTextOfTheBestChunkOfAMergedResult(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Logs/Storm Log.md", 0, 500, "The tide and the lighthouse."),
		host.SearchResult{ChunkID: "c2", Path: "Logs/Storm Log.md", Start: 500, Length: 500, Text: "The tide at the quay.", MatchLine: 7},
		chunk("c3", "Logs/Storm Log.md", 1000, 500, "A tide, a tide, a storm."),
	)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	if len(out.Results) != 1 {
		t.Fatalf("results = %+v, want one merged result", out.Results)
	}
	merged := out.Results[0]
	if merged.Text != "The tide at the quay." || merged.Line != 8 || !isScore(merged.Score, 1) || merged.Start != 0 || merged.Length != 1500 {
		t.Errorf("merged result = %+v, want the text, line and score of the second chunk over the range of all three", merged)
	}
}

func TestPreviewReturnsTheSameResultsTwice(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Logs/B.md", 0, 400, "The tide and the storm."),
		chunk("c2", "Logs/A.md", 0, 400, "The tide and the harbor."),
		chunk("c3", "Logs/C.md", 0, 400, "The storm at the harbor."),
		chunk("c4", "Logs/A.md", 2000, 400, "The tide, the storm and the harbor."),
		chunk("c5", "Logs/D.md", 0, 400, "The harbor at dawn."),
	)
	input := api.PreviewInput{Query: `meaning("tides") || meaning("storms") || words("harbor")`}

	first, second := w.run(t, input), w.run(t, input)

	if !reflect.DeepEqual(first, second) || len(first.Results) != 5 {
		t.Errorf("first run %+v, second run %+v, want five equal results", first.Results, second.Results)
	}
}

func TestPreviewOrdersEqualRoundedScoresByPathThenStart(t *testing.T) {
	w := newWorld(t)
	near := func(cosine float64) []float32 {
		return []float32{float32(cosine), float32(math.Sqrt(1 - cosine*cosine))}
	}
	vectors := map[string][]float32{
		"tides":                 {1, 0},
		"The first passage.":    near(0.71231),
		"The second passage.":   near(0.71229),
		"The third passage.":    near(0.71229),
		"The strongest passage": near(0.9),
	}
	w.search.SetEmbedder(func(text string) []float32 { return vectors[text] })
	w.index(
		chunk("c1", "Notes/b.md", 0, 400, "The first passage."),
		chunk("c2", "Notes/a.md", 5000, 400, "The second passage."),
		chunk("c3", "Notes/a.md", 0, 400, "The third passage."),
		chunk("c4", "Notes/z.md", 0, 400, "The strongest passage"),
	)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	if got := getTexts(out); !slices.Equal(got, []string{"The strongest passage", "The third passage.", "The second passage.", "The first passage."}) {
		t.Errorf("results = %v, want 0.9 first, then the three of 0.712 by path and start", got)
	}
	for _, result := range out.Results[1:] {
		if !isScore(result.Score, 0.712) {
			t.Errorf("result %+v, want the score rounded to 0.712", result)
		}
	}
}

func TestPreviewOrdersWordsOnlyResultsByWordScore(t *testing.T) {
	w := newWorld(t)
	score := func(value float64) *float64 { return &value }
	w.index(
		host.SearchResult{ChunkID: "c1", Path: "Logs/A.md", Start: 0, Length: 400, Text: "The harbor at dawn.", WordScore: score(2.5)},
		host.SearchResult{ChunkID: "c2", Path: "Logs/B.md", Start: 0, Length: 400, Text: "The harbor, the harbor wall.", WordScore: score(7.1)},
		host.SearchResult{ChunkID: "c3", Path: "Logs/C.md", Start: 0, Length: 400, Text: "A harbor seal.", WordScore: score(4.0)},
	)

	out := w.run(t, api.PreviewInput{Query: `words("harbor")`})

	if got := getPaths(out); !slices.Equal(got, []string{"Logs/B.md", "Logs/C.md", "Logs/A.md"}) || out.Order != api.OrderFused {
		t.Errorf("results = %v in order %q, want them by word score, fused", got, out.Order)
	}
	for _, result := range out.Results {
		if result.Score != nil {
			t.Errorf("result %+v, want no score for a result found by words alone", result)
		}
	}
}

func TestPreviewOrdersResultsWithAScoreBeforeResultsWithout(t *testing.T) {
	w := newWorld(t)
	score := func(value float64) *float64 { return &value }
	w.index(
		host.SearchResult{ChunkID: "c1", Path: "Logs/A.md", Start: 0, Length: 400, Text: "The harbor wall.", WordScore: score(9)},
		host.SearchResult{ChunkID: "c2", Path: "Logs/B.md", Start: 0, Length: 400, Text: "The tide tables."},
	)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides") || words("harbor")`})

	if got := getPaths(out); !slices.Equal(got, []string{"Logs/B.md", "Logs/A.md"}) {
		t.Errorf("results = %v, want the result with a similarity first", got)
	}
}
