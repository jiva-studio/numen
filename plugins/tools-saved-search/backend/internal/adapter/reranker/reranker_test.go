package reranker_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/internal/adapter/reranker"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/internal/adapter/reranker/rerankertest"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ports"
)

const (
	tideText   = "tide"
	keeperText = "the harbor master"
)

var _ ports.Reranker = (*reranker.Reranker)(nil)

func newRequest(documents ...string) ports.RerankRequest {
	return ports.RerankRequest{Query: "spring tides", Instruction: "", Documents: documents, Priority: connection.PriorityBackground}
}

func TestRerankerTriesTheRowsInOrderAndNamesTheOneThatAnswered(t *testing.T) {
	rows := rerankertest.NewRows()
	failing := rerankertest.NewStandIn(nil)
	failing.Fail(connection.NewTryNextError(connection.ReasonRateLimited, 0))
	answering := rerankertest.NewStandIn(map[string]float32{tideText: 0.5, keeperText: 0.25})
	rows.Add("conn-quay", "Quay reranker", map[string]string{"model": "quay-model"}, failing)
	rows.Add("conn-pier", "Pier reranker", map[string]string{"model": "pier-model"}, answering)

	reranked, err := reranker.New(rows).Rerank(t.Context(), newRequest(tideText, keeperText))
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(reranked.Scores, []float64{0.5, 0.25}) {
		t.Errorf("scores = %v, want 0.5 and 0.25 from the second row", reranked.Scores)
	}
	if reranked.Reranker.ConnectionID != "conn-pier" || !maps.Equal(reranked.Reranker.Options, map[string]string{"model": "pier-model"}) {
		t.Errorf("reranker = %+v, want the row of conn-pier with its model", reranked.Reranker)
	}
	preferred := rows.GetPreferredRows()
	if len(preferred) != 2 || preferred[0].ConnectionID != "conn-quay" || preferred[1].ConnectionID != "conn-pier" ||
		!preferred[0].IsExclusive || !preferred[1].IsExclusive {
		t.Errorf("preferred rows = %+v, want conn-quay then conn-pier, each exclusive", preferred)
	}
	if got := rows.GetOperations(); !slices.Equal(got, []string{llm.OperationRerank}) {
		t.Errorf("listed operations = %v, want llm.rerank once", got)
	}
	requests := answering.GetRequests()
	if len(requests) != 1 || requests[0].Query != "spring tides" || !slices.Equal(requests[0].Documents, []string{tideText, keeperText}) {
		t.Errorf("requests = %+v, want the query and both documents", requests)
	}
}

func TestRerankerReturnsErrNoRerankerWithoutRows(t *testing.T) {
	_, err := reranker.New(rerankertest.NewRows()).Rerank(t.Context(), newRequest(tideText))

	if !errors.Is(err, ports.ErrNoReranker) {
		t.Errorf("Rerank = %v, want ports.ErrNoReranker", err)
	}
}

func TestRerankerReturnsTheFailuresOfEveryRow(t *testing.T) {
	rows := rerankertest.NewRows()
	quay := rerankertest.NewStandIn(nil)
	quay.Fail(connection.NewTryNextError(connection.ReasonRateLimited, 0))
	pier := rerankertest.NewStandIn(nil)
	pier.Fail(connection.NewTryNextError(connection.ReasonUnavailable, 0))
	rows.Add("conn-quay", "Quay reranker", nil, quay)
	rows.Add("conn-pier", "Pier reranker", nil, pier)

	_, err := reranker.New(rows).Rerank(t.Context(), newRequest(tideText))

	if err == nil || errors.Is(err, ports.ErrNoReranker) {
		t.Fatalf("Rerank = %v, want the failures of both rows", err)
	}
	if !strings.Contains(err.Error(), "Quay reranker") || !strings.Contains(err.Error(), "Pier reranker") {
		t.Errorf("error = %q, want both connection names", err)
	}
}

func TestRerankerStopsAtACanceledContext(t *testing.T) {
	rows := rerankertest.NewRows()
	quay := rerankertest.NewStandIn(nil)
	rows.Add("conn-quay", "Quay reranker", nil, quay)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := reranker.New(rows).Rerank(ctx, newRequest(tideText))

	if !errors.Is(err, context.Canceled) || len(quay.GetRequests()) != 0 {
		t.Errorf("Rerank = %v with %d requests, want context.Canceled and no request", err, len(quay.GetRequests()))
	}
}

func TestListRerankersReturnsTheRowsInOrder(t *testing.T) {
	rows := rerankertest.NewRows()
	rows.Add("conn-quay", "Quay reranker", map[string]string{"model": "quay-model"}, rerankertest.NewStandIn(nil))
	rows.Add("conn-pier", "Pier reranker", map[string]string{"model": "pier-model"}, rerankertest.NewStandIn(nil))

	listed, err := reranker.New(rows).ListRerankers(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []ports.RerankerID{
		{ConnectionID: "conn-quay", Options: map[string]string{"model": "quay-model"}},
		{ConnectionID: "conn-pier", Options: map[string]string{"model": "pier-model"}},
	}
	if len(listed) != len(want) {
		t.Fatalf("rerankers = %+v, want %+v", listed, want)
	}
	for i := range want {
		if listed[i].ConnectionID != want[i].ConnectionID || !maps.Equal(listed[i].Options, want[i].Options) {
			t.Errorf("rerankers[%d] = %+v, want %+v", i, listed[i], want[i])
		}
	}
	if got := rows.GetOperations(); !slices.Equal(got, []string{llm.OperationRerank}) {
		t.Errorf("listed operations = %v, want llm.rerank", got)
	}
}
