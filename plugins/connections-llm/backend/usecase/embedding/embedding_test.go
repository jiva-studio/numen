package embedding_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/embedding"
)

type gateway struct {
	limit   time.Duration
	err     error
	vectors [][]float32
	texts   []string
}

func (g *gateway) Complete(context.Context, connection.Call, time.Duration, connection.ProviderCompleteRequest) (connection.CompleteResult, error) {
	return connection.CompleteResult{}, nil
}

func (g *gateway) Chat(context.Context, connection.Call, time.Duration, connection.ProviderChatRequest, func(connection.ChatEvent) error) error {
	return nil
}

func (g *gateway) Embed(_ context.Context, _ connection.Call, limit time.Duration, texts []string) ([][]float32, error) {
	g.limit, g.texts = limit, texts
	return g.vectors, g.err
}

func (g *gateway) ListModels(context.Context, connection.Call, time.Duration, string) ([]connection.Model, error) {
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
	return connection.Call{Type: typeID, Model: model, Fields: map[string]string{"baseUrl": address}}
}

func TestTextsAreEmbeddedWithTheLimitOfTheOperation(t *testing.T) {
	g := &gateway{vectors: [][]float32{{1, 2}}}

	vectors, err := newEmbedding(t, g).Embed(t.Context(), call("openai", "m", "https://api.test/v1"), []string{"a"})

	if err != nil || len(vectors) != 1 || g.limit != limits.DefaultOtherLimit {
		t.Errorf("vectors = %v, err = %v, limit = %v", vectors, err, g.limit)
	}
}

func TestNoTextIsNoRequest(t *testing.T) {
	g := &gateway{}

	vectors, err := newEmbedding(t, g).Embed(t.Context(), call("openai", "m", "https://api.test/v1"), nil)

	if err != nil || vectors != nil || g.texts != nil {
		t.Errorf("vectors = %v, err = %v, texts = %v", vectors, err, g.texts)
	}
}

func TestTheFirstRequestToACompatibleServerHasALongLimitThatEndsOnceItAnswered(t *testing.T) {
	g := &gateway{vectors: [][]float32{{1}}}
	subject := newEmbedding(t, g)
	local := call("openai-compatible", "m", "http://127.0.0.1:8080/v1")

	if _, err := subject.Embed(t.Context(), local, []string{"a"}); err != nil || g.limit != limits.FirstRequestLimit {
		t.Fatalf("first limit = %v, err = %v", g.limit, err)
	}
	if _, err := subject.Embed(t.Context(), local, []string{"a"}); err != nil || g.limit != limits.DefaultOtherLimit {
		t.Errorf("second limit = %v, err = %v", g.limit, err)
	}
}

func TestAFailedRequestDoesNotEndTheLongFirstLimit(t *testing.T) {
	g := &gateway{err: errors.New("down")}
	subject := newEmbedding(t, g)
	local := call("openai-compatible", "m", "http://127.0.0.1:8080/v1")

	_, _ = subject.Embed(t.Context(), local, []string{"a"})
	_, _ = subject.Embed(t.Context(), local, []string{"a"})

	if g.limit != limits.FirstRequestLimit {
		t.Errorf("limit = %v after failures, want the first limit", g.limit)
	}
}

func TestTheWidthOfTheVectorsIsMeasuredWithOneShortText(t *testing.T) {
	g := &gateway{vectors: [][]float32{{1, 2, 3}}}

	width, err := newEmbedding(t, g).ProbeWidth(t.Context(), call("openai", "m", "https://api.test/v1"))

	if err != nil || width != 3 || len(g.texts) != 1 {
		t.Errorf("width = %d, err = %v, texts = %v", width, err, g.texts)
	}
}

func TestTheProfileOfAKnownModelIsReadFromTheTableWithNoRequest(t *testing.T) {
	g := &gateway{}

	profile, err := newEmbedding(t, g).GetProfile(t.Context(), call("openai", "text-embedding-3-small", "https://api.openai.com/v1/"))

	if err != nil || !profile.IsKnown || profile.Dimensions != 1536 || g.texts != nil {
		t.Errorf("profile = %+v, err = %v", profile, err)
	}
	if profile.Source != "md.numen.connections.llm/openai https://api.openai.com/v1" {
		t.Errorf("source = %q", profile.Source)
	}
}

func TestTheProfileOfAnUnknownModelIsUnknown(t *testing.T) {
	profile, err := newEmbedding(t, &gateway{}).GetProfile(t.Context(), call("openai", "my-model", "https://api.test/v1"))

	if err != nil || profile.IsKnown {
		t.Errorf("profile = %+v, err = %v", profile, err)
	}
}

func TestAProfileOfAnUnknownTypeOrAnAddressThatIsNoURLIsABadRequest(t *testing.T) {
	subject := newEmbedding(t, &gateway{})

	if _, err := subject.GetProfile(t.Context(), call("mystery", "m", "https://api.test/v1")); err == nil {
		t.Error("an unknown type got a profile")
	}
	if _, err := subject.GetProfile(t.Context(), call("openai", "m", "ftp://api.test")); err == nil {
		t.Error("an address that is not http got a profile")
	}
}
