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
	limit       time.Duration
	err         error
	scores      []float32
	isAsked     bool
	query       string
	instruction string
	documents   []string
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

func (g *gateway) Rerank(_ context.Context, _ connection.Call, limit time.Duration, query, instruction string, documents []string) ([]float32, error) {
	g.limit, g.isAsked, g.query, g.instruction, g.documents = limit, true, query, instruction, documents
	return g.scores, g.err
}

func (g *gateway) ListModels(context.Context, connection.Call, time.Duration, []string) ([]connection.Option, error) {
	return nil, nil
}

func newSubject(t *testing.T, g *gateway) *reranking.Reranker {
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
	documents := []string{"Tide tables", "Ferry times"}

	scores, err := newSubject(t, g).Rerank(t.Context(), call("openrouter", "cohere/rerank-v3.5", "https://openrouter.ai/api/v1"), "ferry", "Find the notes that answer the question", documents)

	if err != nil || !slices.Equal(scores, []float32{0.2, 0.9}) || g.limit != limits.DefaultOtherLimit {
		t.Errorf("scores = %v, err = %v, limit = %v", scores, err, g.limit)
	}
	if g.query != "ferry" || g.instruction != "Find the notes that answer the question" || !slices.Equal(g.documents, documents) {
		t.Errorf("the gateway was asked %q, %q, %q", g.query, g.instruction, g.documents)
	}
}

func TestNoDocumentIsNoRequest(t *testing.T) {
	g := &gateway{err: errors.New("should not be called")}
	subject := newSubject(t, g)

	scores, err := subject.Rerank(t.Context(), call("openrouter", "m", "https://openrouter.ai/api/v1"), "ferry", "", nil)

	if err != nil || len(scores) != 0 || g.isAsked {
		t.Errorf("scores = %v, err = %v, asked = %v", scores, err, g.isAsked)
	}

	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := subject.Rerank(canceled, call("openrouter", "m", "https://openrouter.ai/api/v1"), "ferry", "", nil); !errors.Is(err, context.Canceled) {
		t.Errorf("Rerank with canceled context and no document = %v, want context.Canceled", err)
	}
}

func TestARerankingNeedsAGatewayAndLimits(t *testing.T) {
	if _, err := reranking.New(nil, &limits.Limits{}); err == nil {
		t.Error("a reranking was made with no gateway")
	}
	if _, err := reranking.New(&gateway{}, nil); err == nil {
		t.Error("a reranking was made with no limits")
	}
}

func TestTheFirstRerankOfACompatibleModelHasALongLimitThatEndsOnceItAnswered(t *testing.T) {
	g := &gateway{scores: []float32{1}}
	subject := newSubject(t, g)
	local := call("openai-compatible", "bge-reranker-v2-m3", "http://127.0.0.1:8080/v1")
	documents := []string{"Ferry times"}

	if _, err := subject.Rerank(t.Context(), local, "ferry", "", documents); err != nil || g.limit != limits.FirstRequestLimit {
		t.Fatalf("first limit = %v, err = %v", g.limit, err)
	}
	if _, err := subject.Rerank(t.Context(), local, "ferry", "", documents); err != nil || g.limit != limits.DefaultOtherLimit {
		t.Errorf("second limit = %v, err = %v", g.limit, err)
	}
}

func TestAFailedRerankDoesNotEndTheLongFirstLimit(t *testing.T) {
	g := &gateway{err: errors.New("down")}
	subject := newSubject(t, g)
	local := call("openai-compatible", "bge-reranker-v2-m3", "http://127.0.0.1:8080/v1")

	for range 2 {
		if _, err := subject.Rerank(t.Context(), local, "ferry", "", []string{"a"}); err == nil {
			t.Fatal("a failed request returned no error")
		}
	}

	if g.limit != limits.FirstRequestLimit {
		t.Errorf("limit = %v after failures, want the first limit", g.limit)
	}
}
