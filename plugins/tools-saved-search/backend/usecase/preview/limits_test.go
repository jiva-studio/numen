package preview_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/preview"
)

func indexHarborLogs(w *world, count int) {
	chunks := make([]host.SearchResult, count)
	for i := range chunks {
		chunks[i] = chunk(fmt.Sprintf("c%d", i), fmt.Sprintf("Logs/Harbor %03d.md", i), 0, 400, fmt.Sprintf("The harbor log of day %d.", i))
	}
	w.index(chunks...)
}

func TestPreviewReturnsTwentyResultsByDefault(t *testing.T) {
	w := newWorld(t)
	indexHarborLogs(w, 30)

	out := w.run(t, api.PreviewInput{Query: `words("harbor")`})

	if len(out.Results) != 20 {
		t.Errorf("results = %d, want 20", len(out.Results))
	}
}

func TestPreviewCapsALimitAboveOneHundred(t *testing.T) {
	w := newWorld(t)
	w.config.ResultsPerSearch = 300
	indexHarborLogs(w, 150)

	out := w.run(t, api.PreviewInput{Query: `words("harbor")`, Limit: 500})

	if len(out.Results) != 100 {
		t.Errorf("results = %d, want 100", len(out.Results))
	}
}

func TestPreviewUsesTheLimitsOfTheSettings(t *testing.T) {
	w := newWorld(t)
	w.config.DefaultLimit = 5
	w.config.MaxLimit = 8
	indexHarborLogs(w, 30)

	byDefault := w.run(t, api.PreviewInput{Query: `words("harbor")`})
	capped := w.run(t, api.PreviewInput{Query: `words("harbor")`, Limit: 50})

	if len(byDefault.Results) != 5 || len(capped.Results) != 8 {
		t.Errorf("results = %d by default and %d for 50, want 5 and 8", len(byDefault.Results), len(capped.Results))
	}
}

func TestPreviewRejectsANegativeLimit(t *testing.T) {
	w := newWorld(t)
	indexHarborLogs(w, 3)

	if _, err := w.execute(t.Context(), api.PreviewInput{Query: `words("harbor")`, Limit: -1}); err == nil {
		t.Error("Execute accepted the limit -1")
	}
}

func TestPreviewAsksEachSearchForTheResultsPerSearchOfTheSettings(t *testing.T) {
	w := newWorld(t)
	w.config.ResultsPerSearch = 40
	indexHarborLogs(w, 3)

	w.run(t, api.PreviewInput{Query: `words("harbor")`})

	for _, query := range w.host.getQueries() {
		if query.Limit != 40 || query.ResultsPerSource != 40 {
			t.Errorf("search %+v, want a limit of 40 and 40 results of one file", query)
		}
	}
}

func TestPreviewScoresAtMostOneThousandCandidatesInBatchesOfFiveHundred(t *testing.T) {
	w := newWorld(t)
	w.config.ResultsPerSearch = 500
	var chunks []host.SearchResult
	for i, word := range []string{"tide", "storm", "harbor"} {
		for j := range 600 {
			id := fmt.Sprintf("c%d-%d", i, j)
			chunks = append(chunks, chunk(id, fmt.Sprintf("Logs/%s %03d.md", word, j), 0, 400, "A "+word+" log."))
		}
	}
	w.index(chunks...)

	out := w.run(t, api.PreviewInput{Query: `words("tide") || words("storm") || words("harbor")`})

	scored := w.host.getScoredChunks()
	sizes := make([]int, len(scored))
	for i, ids := range scored {
		sizes[i] = len(ids)
	}
	if !slices.Equal(sizes, []int{500, 500}) {
		t.Errorf("scored batches = %v, want two batches of 500", sizes)
	}
	if !hasWarning(out, "the searches found more than 1000 passages; the query was applied to the first 1000") {
		t.Errorf("warnings = %v, want the cut named", out.Warnings)
	}
	tides, storms := 0, 0
	for _, ids := range scored {
		for _, id := range ids {
			switch id[:3] {
			case "c0-":
				tides++
			case "c1-":
				storms++
			}
		}
	}
	if tides < 300 || storms < 300 {
		t.Errorf("scored %d tide and %d storm chunks, want the union taken in turns from every search", tides, storms)
	}
}

func TestPreviewRejectsAQueryOverTheMaximumBytes(t *testing.T) {
	w := newWorld(t)
	indexHarborLogs(w, 3)
	long := `words("harbor")` + strings.Repeat(" ", 16384)

	out, err := w.execute(t.Context(), api.PreviewInput{Query: long})
	if err != nil || len(out.Errors) != 1 || !strings.Contains(out.Errors[0].Message, "a query has at most 16384 bytes") {
		t.Errorf("Execute = %+v, %v, want the query rejected for its size", out.Errors, err)
	}

	w.config.MaxQueryBytes = 32768
	if allowed := w.run(t, api.PreviewInput{Query: long}); len(allowed.Results) != 3 {
		t.Errorf("results = %d under a larger maximum, want 3", len(allowed.Results))
	}
}

func TestPreviewRejectsMoreThanEightPathPrefixes(t *testing.T) {
	w := newWorld(t)
	indexHarborLogs(w, 3)
	prefixes := make([]string, 9)
	for i := range prefixes {
		prefixes[i] = fmt.Sprintf(`path.startsWith("Folder %d/")`, i)
	}
	text := `words("harbor") && (` + strings.Join(prefixes, " || ") + `)`

	out, err := w.execute(t.Context(), api.PreviewInput{Query: text})
	if err != nil || len(out.Errors) == 0 {
		t.Errorf("Execute = %+v, %v, want nine prefixes rejected", out, err)
	}

	w.config.MaxPathPrefixes = 9
	w.run(t, api.PreviewInput{Query: text})
	if got := len(w.host.getQueries()); got != 9 {
		t.Errorf("searches = %d under a maximum of 9 prefixes, want one in each folder", got)
	}
}

func TestDefaultConfigContainsTheNumbersOfTheOwner(t *testing.T) {
	want := preview.Config{
		DefaultLimit: 20, MaxLimit: 100, ResultsPerSearch: 100, RerankResultsPerSearch: 300, MaxCandidates: 1000,
		MaxPathPrefixes: 8, FusedPerResult: 3, MaxFused: 100, RerankedPerResult: 10, MaxReranked: 300, MaxRerankedBytes: 4096,
		MergeGapBytes: 200, MaxQueryBytes: 16384,
	}

	if got := preview.DefaultConfig(); got != want {
		t.Errorf("DefaultConfig() = %+v, want %+v", got, want)
	}
}

func TestParseConfigReadsEveryNumberOfTheSettings(t *testing.T) {
	got := preview.ParseConfig(map[string]string{
		"defaultLimit": "10", "maxLimit": "50", "resultsPerSearch": "200", "maxCandidates": "800", "maxPathPrefixes": "4",
		"rerankResultsPerSearch": "400", "fusedPerResult": "4", "maxFused": "80", "rerankedPerResult": "2", "maxReranked": "60",
		"maxRerankedBytes": "2048", "mergeGapBytes": "0", "maxQueryBytes": "4096",
	}, preview.DefaultConfig())

	want := preview.Config{
		DefaultLimit: 10, MaxLimit: 50, ResultsPerSearch: 200, RerankResultsPerSearch: 400, MaxCandidates: 800,
		MaxPathPrefixes: 4, FusedPerResult: 4, MaxFused: 80, RerankedPerResult: 2, MaxReranked: 60, MaxRerankedBytes: 2048,
		MergeGapBytes: 0, MaxQueryBytes: 4096,
	}
	if got != want {
		t.Errorf("ParseConfig = %+v, want %+v", got, want)
	}
}

func TestParseConfigKeepsANumberThatIsNotValid(t *testing.T) {
	got := preview.ParseConfig(map[string]string{
		"defaultLimit": "many", "maxLimit": "0", "resultsPerSearch": "-3", "mergeGapBytes": "-1", "maxQueryBytes": "",
	}, preview.DefaultConfig())

	if got != preview.DefaultConfig() {
		t.Errorf("ParseConfig = %+v, want the defaults", got)
	}
}

func TestPreviewReranksAtMostTheMaximumOfTheSettings(t *testing.T) {
	w := newWorld(t)
	w.config.MaxReranked = 4
	indexHarborLogs(w, 30)
	standIn := w.withReranker(nil)

	w.run(t, api.PreviewInput{Query: `words("harbor")`, Limit: 10})

	if requests := standIn.getRequests(); len(requests) != 1 || len(requests[0].GetDocuments()) != 4 {
		t.Errorf("rerank requests = %v, want one of 4 documents", requests)
	}
}

func TestPreviewAsksEachSearchForMorePassagesAndTheirWholeTextWithAConnection(t *testing.T) {
	w := newWorld(t)
	indexHarborLogs(w, 3)
	w.withReranker(nil)

	w.run(t, api.PreviewInput{Query: `words("harbor") || meaning("harbor")`})

	queries := w.host.getQueries()
	if len(queries) != 2 {
		t.Fatalf("searches = %+v, want one for each condition", queries)
	}
	for _, query := range queries {
		if query.Limit != 300 || query.ResultsPerSource != 300 || !query.IsWholeText {
			t.Errorf("search %+v, want a limit of 300, 300 results of one file and the whole text", query)
		}
	}
}

func TestPreviewAsksEachSearchForTheExcerptsAloneWithoutAConnection(t *testing.T) {
	w := newWorld(t)
	indexHarborLogs(w, 3)

	w.run(t, api.PreviewInput{Query: `words("harbor")`})

	for _, query := range w.host.getQueries() {
		if query.Limit != 100 || query.IsWholeText {
			t.Errorf("search %+v, want a limit of 100 and no whole text", query)
		}
	}
}
