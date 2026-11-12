package embedding_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/embedding"
)

type gateway struct {
	limit   time.Duration
	err     error
	vectors [][]float32
	texts   []string
}

func (g *gateway) Complete(context.Context, connection.Call, time.Duration, *llm.CompleteRequest) (*llm.CompleteResult, error) {
	return &llm.CompleteResult{}, nil
}

func (g *gateway) Chat(context.Context, connection.Call, time.Duration, *llm.ChatRequest, func(*llm.ChatFrame) error) (*llm.ChatFinish, error) {
	return &llm.ChatFinish{}, nil
}

func (g *gateway) Embed(_ context.Context, _ connection.Call, limit time.Duration, texts []string) ([][]float32, error) {
	g.limit, g.texts = limit, texts
	return g.vectors, g.err
}

func (g *gateway) ListModels(context.Context, connection.Call, time.Duration, []string) ([]connection.Option, error) {
	return nil, nil
}

func newEmbedding(t *testing.T, g *gateway) *embedding.Embedding {
	t.Helper()
	subject, err := embedding.New(g, &limits.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	return subject
}

func call(typeID, model, address string) connection.Call {
	return connection.Call{Type: typeID, Options: map[string]string{llm.OptionModel: model}, Fields: map[string]string{"baseUrl": address}}
}

func TestTextsAreEmbeddedWithTheLimitOfTheOperation(t *testing.T) {
	g := &gateway{vectors: [][]float32{{1, 2}}}

	result, err := newEmbedding(t, g).Embed(t.Context(), call("openai", "m", "https://api.test/v1"), &llm.EmbedRequest{Texts: []string{"a"}})

	if err != nil || len(result.GetVectors()) != 1 || g.limit != limits.DefaultOtherLimit {
		t.Errorf("result = %v, err = %v, limit = %v", result, err, g.limit)
	}
}

func TestNoTextIsNoRequest(t *testing.T) {
	g := &gateway{}

	result, err := newEmbedding(t, g).Embed(t.Context(), call("openai", "m", "https://api.test/v1"), &llm.EmbedRequest{})

	if err != nil || len(result.GetVectors()) != 0 || g.texts != nil {
		t.Errorf("result = %v, err = %v, texts = %v", result, err, g.texts)
	}
}

func TestTheFirstRequestToACompatibleServerHasALongLimitThatEndsOnceItAnswered(t *testing.T) {
	g := &gateway{vectors: [][]float32{{1}}}
	subject := newEmbedding(t, g)
	local := call("openai-compatible", "m", "http://127.0.0.1:8080/v1")

	if _, err := subject.Embed(t.Context(), local, &llm.EmbedRequest{Texts: []string{"a"}}); err != nil || g.limit != limits.FirstRequestLimit {
		t.Fatalf("first limit = %v, err = %v", g.limit, err)
	}
	if _, err := subject.Embed(t.Context(), local, &llm.EmbedRequest{Texts: []string{"a"}}); err != nil || g.limit != limits.DefaultOtherLimit {
		t.Errorf("second limit = %v, err = %v", g.limit, err)
	}
}

func TestAFailedRequestDoesNotEndTheLongFirstLimit(t *testing.T) {
	g := &gateway{err: errors.New("down")}
	subject := newEmbedding(t, g)
	local := call("openai-compatible", "m", "http://127.0.0.1:8080/v1")

	for range 2 {
		if _, err := subject.Embed(t.Context(), local, &llm.EmbedRequest{Texts: []string{"a"}}); err == nil {
			t.Fatal("a failed request returned no error")
		}
	}

	if g.limit != limits.FirstRequestLimit {
		t.Errorf("limit = %v after failures, want the first limit", g.limit)
	}
}

func TestTheWidthOfAModelOutsideTheTableIsMeasuredWithOneShortText(t *testing.T) {
	g := &gateway{vectors: [][]float32{{1, 2, 3}}}

	profile, err := newEmbedding(t, g).GetProfile(t.Context(), call("openai", "my-model", "https://api.test/v1"))

	if err != nil || profile.GetWidth() != 3 || profile.GetModel() != "my-model" || profile.GetPooling() != "" || len(g.texts) != 1 {
		t.Errorf("profile = %v, err = %v, texts = %v", profile, err, g.texts)
	}
}

func TestTheProfileOfAKnownModelIsReadFromTheTableWithNoRequest(t *testing.T) {
	g := &gateway{}

	profile, err := newEmbedding(t, g).GetProfile(t.Context(), call("openai", "text-embedding-3-small", "https://api.openai.com/v1/"))

	if err != nil || profile.GetWidth() != 1536 || profile.GetPooling() == "" || g.texts != nil {
		t.Errorf("profile = %v, err = %v", profile, err)
	}
	if profile.GetSource() != "md.numen.connections.llm/openai https://api.openai.com/v1" {
		t.Errorf("source = %q", profile.GetSource())
	}
}

func TestTheProfileOfAModelOutsideTheTableFailsAsTheMeasureFails(t *testing.T) {
	failure := connection.NewTryNextError(connection.ReasonUnavailable, 0)

	_, err := newEmbedding(t, &gateway{err: failure}).GetProfile(t.Context(), call("openai", "my-model", "https://api.test/v1"))

	if !errors.Is(err, failure) {
		t.Errorf("err = %v, want the failure of the service", err)
	}
}

func TestAProfileOfAnUnknownTypeOrAnAddressThatIsNoURLIsAnInvalidRequest(t *testing.T) {
	subject := newEmbedding(t, &gateway{})

	if _, err := subject.GetProfile(t.Context(), call("mystery", "m", "https://api.test/v1")); err == nil {
		t.Error("an unknown type got a profile")
	}
	if _, err := subject.GetProfile(t.Context(), call("openai", "m", "ftp://api.test")); err == nil {
		t.Error("an address that is not http got a profile")
	}
}
