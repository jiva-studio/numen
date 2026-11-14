package preview_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
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

func TestPreviewReranksTheFirstThreeTimesTheLimitInFusedOrder(t *testing.T) {
	w := newWorld(t)
	var texts []string
	var chunks []host.SearchResult
	for i := range 8 {
		text := strings.TrimSpace(strings.Repeat("tide ", 9-i)) + " storm"
		texts = append(texts, text)
		chunks = append(chunks, chunk(fmt.Sprintf("c%d", i), fmt.Sprintf("Logs/%d.md", i), 0, 400, text))
	}
	w.index(chunks...)
	standIn := w.withReranker(nil)

	out := w.run(t, api.PreviewInput{Query: `meaning("tides")`, Limit: 2})

	requests := standIn.getRequests()
	if len(requests) != 1 || !slices.Equal(requests[0].GetDocuments(), texts[:6]) {
		t.Errorf("rerank requests = %v, want the first six texts in fused order", requests)
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
