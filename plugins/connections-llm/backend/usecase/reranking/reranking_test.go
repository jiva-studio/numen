package reranking_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/reranking"
)

type gateway struct {
	limit   time.Duration
	err     error
	scores  []float32
	request *llm.RerankRequest
}

func (g *gateway) Complete(context.Context, connection.Call, time.Duration, *llm.CompleteRequest) (*llm.CompleteResult, error) {
	return &llm.CompleteResult{}, nil
}

func (g *gateway) Chat(context.Context, connection.Call, time.Duration, *llm.ChatRequest, func(*llm.ChatFrame) error) (*llm.ChatFinish, error) {
	return &llm.ChatFinish{}, nil
}

func (g *gateway) Embed(context.Context, connection.Call, time.Duration, []string) ([][]float32, error) {
	return nil, nil
}

func (g *gateway) Rerank(_ context.Context, _ connection.Call, limit time.Duration, request *llm.RerankRequest) ([]float32, error) {
	g.limit, g.request = limit, request
	return g.scores, g.err
}

func (g *gateway) ListModels(context.Context, connection.Call, time.Duration, []string) ([]connection.Option, error) {
	return nil, nil
}

func newSubject(t *testing.T, g *gateway) *reranking.Reranking {
	t.Helper()
	subject, err := reranking.New(g, &limits.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	return subject
}

func call(typeID, model, address string) connection.Call {
	return connection.Call{Type: typeID, Options: map[string]string{llm.OptionModel: model}, Fields: map[string]string{"baseUrl": address}}
}

func TestDocumentsAreRerankedWithTheLimitOfTheOperation(t *testing.T) {
	g := &gateway{scores: []float32{0.2, 0.9}}
	request := &llm.RerankRequest{Query: "ferry", Documents: []string{"Tide tables", "Ferry times"}}

	result, err := newSubject(t, g).Rerank(t.Context(), call("openrouter", "cohere/rerank-v3.5", "https://openrouter.ai/api/v1"), request)

	if err != nil || !slices.Equal(result.GetScores(), []float32{0.2, 0.9}) || g.limit != limits.DefaultOtherLimit {
		t.Errorf("result = %v, err = %v, limit = %v", result, err, g.limit)
	}
	if g.request.GetQuery() != "ferry" || !slices.Equal(g.request.GetDocuments(), request.GetDocuments()) {
		t.Errorf("the gateway was asked %v", g.request)
	}
}

func TestNoDocumentIsNoRequest(t *testing.T) {
	g := &gateway{}

	result, err := newSubject(t, g).Rerank(t.Context(), call("openrouter", "m", "https://openrouter.ai/api/v1"), &llm.RerankRequest{Query: "ferry"})

	if err != nil || len(result.GetScores()) != 0 || g.request != nil {
		t.Errorf("result = %v, err = %v, request = %v", result, err, g.request)
	}
}

func TestTheFirstRerankOfACompatibleModelHasALongLimitThatEndsOnceItAnswered(t *testing.T) {
	g := &gateway{scores: []float32{1}}
	subject := newSubject(t, g)
	local := call("openai-compatible", "bge-reranker-v2-m3", "http://127.0.0.1:8080/v1")
	request := &llm.RerankRequest{Query: "ferry", Documents: []string{"Ferry times"}}

	if _, err := subject.Rerank(t.Context(), local, request); err != nil || g.limit != limits.FirstRequestLimit {
		t.Fatalf("first limit = %v, err = %v", g.limit, err)
	}
	if _, err := subject.Rerank(t.Context(), local, request); err != nil || g.limit != limits.DefaultOtherLimit {
		t.Errorf("second limit = %v, err = %v", g.limit, err)
	}
}

func TestAFailedRerankDoesNotEndTheLongFirstLimit(t *testing.T) {
	g := &gateway{err: errors.New("down")}
	subject := newSubject(t, g)
	local := call("openai-compatible", "bge-reranker-v2-m3", "http://127.0.0.1:8080/v1")

	for range 2 {
		if _, err := subject.Rerank(t.Context(), local, &llm.RerankRequest{Query: "ferry", Documents: []string{"a"}}); err == nil {
			t.Fatal("a failed request returned no error")
		}
	}

	if g.limit != limits.FirstRequestLimit {
		t.Errorf("limit = %v after failures, want the first limit", g.limit)
	}
}
