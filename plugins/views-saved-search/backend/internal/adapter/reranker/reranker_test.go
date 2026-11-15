package reranker_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/internal/adapter/reranker"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ports"
)

// byLength scores a document by the number of its bytes, divided by eight, and records the request.
type byLength struct {
	request *llm.RerankRequest
	err     error
}

func (b *byLength) Rerank(_ context.Context, _ connection.Call, request *llm.RerankRequest) (*llm.RerankResult, error) {
	b.request = request
	if b.err != nil {
		return nil, b.err
	}
	result := &llm.RerankResult{}
	for _, document := range request.GetDocuments() {
		result.Scores = append(result.Scores, float32(len(document))/8)
	}
	return result, nil
}

func newReranker(model llm.Reranker) *reranker.Reranker {
	return reranker.New(llmtest.NewCaller(llm.Provide(nil, llm.Roles{Reranker: model}), connection.Call{ConnectionID: "harbor-reranker"}))
}

var _ ports.Reranker = (*reranker.Reranker)(nil)

func TestRerankerReturnsTheScoresOfTheService(t *testing.T) {
	model := &byLength{}

	scores, err := newReranker(model).Rerank(t.Context(), "spring tides", "", []string{"tide", "the harbor master"})

	if err != nil || !slices.Equal(scores, []float64{0.5, 2.125}) {
		t.Errorf("Rerank = %v, %v, want 0.5 and 2.125", scores, err)
	}
	if model.request.GetQuery() != "spring tides" || model.request.GetInstruction() != "" || len(model.request.GetDocuments()) != 2 {
		t.Errorf("request = %v, want the query, no instruction and both documents", model.request)
	}
}

func TestRerankerReturnsErrNoRerankerWithoutAConnection(t *testing.T) {
	caller := llmtest.NewCaller(llm.Provide(nil, llm.Roles{}), connection.Call{})

	_, err := reranker.New(caller).Rerank(t.Context(), "spring tides", "", []string{"tide"})

	if !errors.Is(err, ports.ErrNoReranker) {
		t.Errorf("Rerank = %v, want ports.ErrNoReranker", err)
	}
}

func TestRerankerPassesAFailureOfTheServiceThrough(t *testing.T) {
	model := &byLength{err: connection.NewTryNextError(connection.ReasonRateLimited, 0)}

	_, err := newReranker(model).Rerank(t.Context(), "spring tides", "", []string{"tide"})

	var failed *connection.AllFailedError
	if !errors.As(err, &failed) || errors.Is(err, ports.ErrNoReranker) {
		t.Errorf("Rerank = %v, want the failure of the service", err)
	}
}
