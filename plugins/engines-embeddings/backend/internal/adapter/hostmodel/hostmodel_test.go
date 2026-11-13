package hostmodel

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

type askedCall struct {
	operation string
	priority  connection.Priority
}

// fixedModel answers llm.embed with vectors and llm.profile with profile.
type fixedModel struct {
	vectors *llm.EmbedResult
	profile *llm.Profile
}

func (m fixedModel) Embed(context.Context, connection.Call, *llm.EmbedRequest) (*llm.EmbedResult, error) {
	return m.vectors, nil
}

func (m fixedModel) GetProfile(context.Context, connection.Call) (*llm.Profile, error) {
	return m.profile, nil
}

// fakeCaller asks the connection of model for each operation, or fails every call with err.
type fakeCaller struct {
	model fixedModel
	err   error
	asked []askedCall
}

func (f *fakeCaller) Call(ctx context.Context, operation string, payload []byte, priority connection.Priority) (connection.Frames, error) {
	f.asked = append(f.asked, askedCall{operation: operation, priority: priority})
	if f.err != nil {
		return nil, f.err
	}
	provider := llm.Provide(nil, llm.Roles{Embedder: f.model, Profiler: f.model})
	return llmtest.NewCaller(provider, connection.Call{}).Call(ctx, operation, payload, priority)
}

func newCaller() *fakeCaller {
	return &fakeCaller{model: fixedModel{vectors: &llm.EmbedResult{Vectors: []*llm.Vector{{Values: []float32{1}}}}}}
}

func TestQueriesAreInteractiveAndDocumentsAreBackground(t *testing.T) {
	caller := newCaller()
	models := New(caller)

	if _, err := models.Embed(t.Context(), model.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := models.Embed(t.Context(), model.RoleQuery, []string{"b"}); err != nil {
		t.Fatal(err)
	}

	if caller.asked[0].priority != connection.PriorityBackground || caller.asked[1].priority != connection.PriorityInteractive {
		t.Errorf("priorities = %v and %v", caller.asked[0].priority, caller.asked[1].priority)
	}
	if caller.asked[0].operation != llm.OperationEmbed {
		t.Errorf("operation = %q, want %q", caller.asked[0].operation, llm.OperationEmbed)
	}
}

func TestTheProfileOfTheHostIsReadAsAConnectionProfile(t *testing.T) {
	caller := &fakeCaller{model: fixedModel{profile: &llm.Profile{
		Model: "m", Source: "type https://example.org", Pooling: "cls", QueryPrefix: "q: ", DocumentPrefix: "d: ", Width: 8, MaxTokens: 512,
	}}}

	got, err := New(caller).GetEmbeddingProfile(t.Context())

	want := model.ConnectionProfile{Model: "m", Source: "type https://example.org", Pooling: "cls", QueryPrefix: "q: ", DocumentPrefix: "d: ", Dimensions: 8, MaxTokens: 512}
	if err != nil || got != want {
		t.Errorf("profile = %+v, %v, want %+v", got, err, want)
	}
	if caller.asked[0].operation != llm.OperationProfile || caller.asked[0].priority != connection.PriorityBackground {
		t.Errorf("asked %+v, want llm.profile in the background", caller.asked[0])
	}
}

func TestFailuresOfTheConnectionsAreConnectionErrors(t *testing.T) {
	cases := map[string]error{
		"every row failed": &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "OpenRouter", Reason: "payment required"}}},
		"a row stopped":    &connection.StoppedError{ConnectionName: "Ollama", Reason: "key rejected"},
		"no row":           connection.ErrNoConnection,
	}
	for name, failure := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := New(&fakeCaller{err: failure}).Embed(t.Context(), model.RoleDocument, []string{"a"})

			if failure, isConnection := errors.AsType[ports.ConnectionError](err); !isConnection || failure.Err == nil {
				t.Errorf("err = %v, want a connection error", err)
			}
		})
	}
}

func TestNoRowAsksForAConnectionInPlainWords(t *testing.T) {
	_, err := New(&fakeCaller{err: connection.ErrNoConnection}).GetEmbeddingProfile(t.Context())

	if !errors.Is(err, ports.ErrNoConnection) || err.Error() != "choose a connection in the settings of Vector Embeddings" {
		t.Errorf("err = %v, want the request to choose a connection", err)
	}
}

func TestACanceledCallKeepsItsContextError(t *testing.T) {
	_, err := New(&fakeCaller{err: context.Canceled}).Embed(t.Context(), model.RoleDocument, []string{"a"})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the cancellation", err)
	}
	if failure, isConnection := errors.AsType[ports.ConnectionError](err); isConnection && failure.Err != nil {
		t.Error("a cancellation was reported as a connection error")
	}
}
