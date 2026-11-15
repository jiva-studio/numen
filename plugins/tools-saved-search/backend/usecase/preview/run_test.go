package preview_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/internal/adapter/reranker/rerankertest"
)

func TestRunReturnsTheChunkIDsOfEachResult(t *testing.T) {
	w := newWorld(t)
	w.index(
		chunk("c1", "Logs/Gale.md", 0, 400, "A storm at the quay."),
		chunk("c2", "Logs/Gale.md", 400, 400, "The storm broke a mooring."),
		chunk("c3", "Logs/Calm.md", 0, 400, "A storm passed far out."),
	)

	run, err := w.newUseCase().Run(t.Context(), api.PreviewInput{Query: `meaning("storm")`})
	if err != nil {
		t.Fatal(err)
	}

	if len(run.Output.Results) != 2 || len(run.ChunkIDs) != 2 {
		t.Fatalf("run = %+v, want two results and their chunk ids", run)
	}
	for i, result := range run.Output.Results {
		want := map[string][]string{"Logs/Gale.md": {"c1", "c2"}, "Logs/Calm.md": {"c3"}}[result.Path]
		if !slices.Equal(run.ChunkIDs[i], want) {
			t.Errorf("chunk ids of %s = %v, want %v in start order", result.Path, run.ChunkIDs[i], want)
		}
	}
	if run.Reranker != nil || run.Rerank != "storm" {
		t.Errorf("run = %+v, want no reranker in fused order and the leaf text as the rerank query", run)
	}
}

func TestRunNamesTheRowThatReranked(t *testing.T) {
	w := newWorld(t)
	indexThreeTides(w)
	failing := rerankertest.NewStandIn(nil)
	failing.Fail(connection.NewTryNextError(connection.ReasonUnavailable, 0))
	w.rows.Add("conn-quay", "Quay reranker", map[string]string{"model": "quay-model"}, failing)
	w.rows.Add("conn-pier", "Pier reranker", map[string]string{"model": "pier-model"},
		rerankertest.NewStandIn(map[string]float32{highTide: 0.1, midTide: 0.5, lowTide: 0.9}))

	run, err := w.newUseCase().Run(t.Context(), api.PreviewInput{Query: `meaning("tides")`, Rerank: "spring tides at the quay"})
	if err != nil {
		t.Fatal(err)
	}

	if run.Output.Order != api.OrderReranked || !slices.Equal(getTexts(run.Output), []string{lowTide, midTide, highTide}) {
		t.Errorf("output = %+v, want the order of the second row", run.Output)
	}
	if run.Reranker == nil || run.Reranker.ConnectionID != "conn-pier" || run.Reranker.Options["model"] != "pier-model" {
		t.Errorf("reranker = %+v, want the row of conn-pier", run.Reranker)
	}
	if run.Rerank != "spring tides at the quay" {
		t.Errorf("rerank = %q, want the rerank text sent", run.Rerank)
	}
}

func TestExecuteReturnsTheOutputOfRun(t *testing.T) {
	w := newWorld(t)
	indexThreeTides(w)
	w.withReranker(map[string]float32{highTide: 0.1, midTide: 0.5, lowTide: 0.9})
	input := api.PreviewInput{Query: `meaning("tides")`}

	run, err := w.newUseCase().Run(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	out := w.run(t, input)

	if !slices.Equal(getTexts(out), getTexts(run.Output)) || out.Order != run.Output.Order || out.Floor != run.Output.Floor {
		t.Errorf("Execute = %+v, Run = %+v, want the same output", out, run.Output)
	}
}
