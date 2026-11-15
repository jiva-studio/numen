package preview_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
)

const (
	highTide = "A tide, a tide, a tide."
	midTide  = "A tide and a storm."
	lowTide  = "A tide, a storm, a storm."
)

func indexThreeTides(w *world) {
	w.index(
		chunk("c1", "Logs/High.md", 0, 400, highTide),
		chunk("c2", "Logs/Mid.md", 0, 400, midTide),
		chunk("c3", "Logs/Low.md", 0, 400, lowTide),
	)
}

func TestPreviewOrdersResultsByTheRerankScores(t *testing.T) {
	w := newWorld(t)
	indexThreeTides(w)
	standIn := w.withReranker(map[string]float32{highTide: 0.1, midTide: 0.5, lowTide: 0.9})

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	if got := getTexts(out); !slices.Equal(got, []string{lowTide, midTide, highTide}) || out.Order != api.OrderReranked {
		t.Errorf("results = %v in order %q, want them by the rerank scores, reranked", got, out.Order)
	}
	if !isScore(out.Results[0].Score, 0.9) || !isScore(out.Results[2].Score, 0.1) {
		t.Errorf("results = %+v, want the rerank scores rounded to 3 decimals", out.Results)
	}
	requests := standIn.getRequests()
	if len(requests) != 1 || !slices.Equal(requests[0].GetDocuments(), []string{highTide, midTide, lowTide}) {
		t.Errorf("rerank requests = %v, want one with the texts in fused order", requests)
	}
}

func TestPreviewSendsTheRerankTextAsTheQueryAndNoInstruction(t *testing.T) {
	w := newWorld(t)
	indexThreeTides(w)
	standIn := w.withReranker(nil)

	w.run(t, api.PreviewInput{Query: `meaning("tides") || words("storm")`, Rerank: "when the spring tides reach the quay"})

	requests := standIn.getRequests()
	if len(requests) != 1 || requests[0].GetQuery() != "when the spring tides reach the quay" || requests[0].GetInstruction() != "" {
		t.Errorf("rerank requests = %v, want the rerank text as the query and no instruction", requests)
	}
}

func TestPreviewSendsThePositiveLeafTextsWithoutARerankText(t *testing.T) {
	w := newWorld(t)
	indexThreeTides(w)
	standIn := w.withReranker(nil)

	w.run(t, api.PreviewInput{Query: `(meaning("tides") || words("storm")) && !meaning("lighthouses")`})

	requests := standIn.getRequests()
	if len(requests) != 1 || requests[0].GetQuery() != "tides\nstorm" || requests[0].GetInstruction() != "" {
		t.Errorf("rerank requests = %v, want the texts of the positive leaves joined by newlines", requests)
	}
}

func TestPreviewReranksTheFirstTenTimesTheLimitInFusedOrder(t *testing.T) {
	w := newWorld(t)
	var texts []string
	var chunks []host.SearchResult
	for i := range 25 {
		text := strings.TrimSpace(strings.Repeat("tide ", 26-i)) + " storm"
		texts = append(texts, text)
		chunks = append(chunks, chunk(fmt.Sprintf("c%d", i), fmt.Sprintf("Logs/%02d.md", i), 0, 400, text))
	}
	w.index(chunks...)
	standIn := w.withReranker(nil)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`, Limit: 2})

	requests := standIn.getRequests()
	if len(requests) != 1 || !slices.Equal(requests[0].GetDocuments(), texts[:20]) {
		t.Errorf("rerank requests = %v, want the first twenty texts in fused order", requests)
	}
	if len(out.Results) != 2 {
		t.Errorf("results = %v, want two", getTexts(out))
	}
}

func TestPreviewKeepsTheFusedOrderWithoutAConnection(t *testing.T) {
	w := newWorld(t)
	indexThreeTides(w)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`, Rerank: "spring tides"})

	if got := getTexts(out); !slices.Equal(got, []string{highTide, midTide, lowTide}) || out.Order != api.OrderFused || len(out.Warnings) != 0 {
		t.Errorf("output = %+v, want the fused order and no warning", out)
	}
}

func TestPreviewKeepsTheFusedOrderAndWarnsWhenTheRerankerFails(t *testing.T) {
	w := newWorld(t)
	indexThreeTides(w)
	standIn := w.withReranker(nil)
	standIn.err = connection.NewTryNextError(connection.ReasonUnavailable, 0)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	if got := getTexts(out); !slices.Equal(got, []string{highTide, midTide, lowTide}) || out.Order != api.OrderFused {
		t.Errorf("results = %v in order %q, want the fused order", got, out.Order)
	}
	if !hasWarning(out, "the reranking service did not answer") || !hasWarning(out, "the results are in fused order") {
		t.Errorf("warnings = %v, want the reranking service named", out.Warnings)
	}
	if !isScore(out.Results[0].Score, 1) {
		t.Errorf("results = %+v, want the similarity as the score", out.Results)
	}
}

func TestPreviewWarnsWhenTheRerankerAnswersAnotherNumberOfScores(t *testing.T) {
	w := newWorld(t)
	indexThreeTides(w)
	standIn := w.withReranker(map[string]float32{lowTide: 0.9})
	standIn.count = 1

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	if out.Order != api.OrderFused || !hasWarning(out, "the reranking service did not answer") {
		t.Errorf("output = %+v, want the fused order and a warning", out)
	}
}

func TestPreviewDoesNotAskTheRerankerWhenNothingMatches(t *testing.T) {
	w := newWorld(t)
	w.index(chunk("c1", "Logs/Keeper.md", 0, 400, "The lighthouse keeper."))
	standIn := w.withReranker(nil)

	out := w.run(t, api.PreviewInput{Query: `words("lighthouse") && meaning("tides")`})

	if len(out.Results) != 0 || len(standIn.getRequests()) != 0 {
		t.Errorf("output = %+v, rerank requests = %v, want no result and no request", out, standIn.getRequests())
	}
}

func TestPreviewSendsTheWholeTextOfEachPassageToTheReranker(t *testing.T) {
	w := newWorld(t)
	long := strings.Repeat("The tide log of the north quay. ", 60) + "At the spring tide the water covers the lower steps."
	w.index(chunk("c1", "Logs/Quay.md", 0, len(long), long))
	standIn := w.withReranker(nil)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	requests := standIn.getRequests()
	if len(requests) != 1 || !slices.Equal(requests[0].GetDocuments(), []string{long}) {
		t.Errorf("rerank requests = %v, want the whole text of the passage", requests)
	}
	if len(out.Results) != 1 || len(out.Results[0].Text) > host.MaxExcerptBytes {
		t.Errorf("results = %+v, want the excerpt as the text of the result", out.Results)
	}
}

func TestPreviewSendsAtMostMaxRerankedBytesOfEachTextToTheReranker(t *testing.T) {
	w := newWorld(t)
	w.config.MaxRerankedBytes = 101
	long := "tide " + strings.Repeat("прилив у причала ", 20)
	w.index(chunk("c1", "Logs/Причал.md", 0, len(long), long))
	standIn := w.withReranker(nil)

	w.run(t, api.PreviewInput{Query: `meaning("tides")`})

	requests := standIn.getRequests()
	if len(requests) != 1 || len(requests[0].GetDocuments()) != 1 {
		t.Fatalf("rerank requests = %v, want one with one document", requests)
	}
	sent := requests[0].GetDocuments()[0]
	if len(sent) > 101 || len(sent) < 98 || !utf8.ValidString(sent) || !strings.HasPrefix(long, sent) {
		t.Errorf("sent %q (%d bytes), want the start of the text cut at a whole character within 101 bytes", sent, len(sent))
	}
}

func TestPreviewJoinsOnlyTheFirstThreeTimesTheLimitWithoutAConnection(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Logs/North.md", 0, 400, "tide tide tide tide tide storm"),
		chunk("c2", "Logs/East.md", 0, 400, "tide tide tide tide storm"),
		chunk("c3", "Logs/South.md", 0, 400, "tide tide tide storm"),
		chunk("c4", "Logs/West.md", 0, 400, "tide tide storm"),
		chunk("c5", "Logs/North.md", 400, 400, "tide storm"),
	)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`, Limit: 1})

	if len(out.Results) != 1 || out.Results[0].Path != "Logs/North.md" || out.Results[0].ChunkCount != 1 {
		t.Errorf("results = %+v, want North alone: its neighbor is fifth in fused order, past three times the limit", out.Results)
	}
}
