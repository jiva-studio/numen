package hostmodel

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

type fakeClient struct {
	profile connection.EmbeddingProfile
	err     error
	asked   []connection.EmbedRequest
}

func (f *fakeClient) Embed(_ context.Context, request connection.EmbedRequest) (connection.EmbedResult, error) {
	f.asked = append(f.asked, request)
	return connection.EmbedResult{Vectors: [][]float32{{1}}}, f.err
}

func (f *fakeClient) GetEmbeddingProfile(context.Context) (connection.EmbeddingProfile, error) {
	return f.profile, f.err
}

func TestQueriesAreInteractiveAndDocumentsAreBackground(t *testing.T) {
	client := &fakeClient{}
	models := New(client)

	if _, err := models.Embed(t.Context(), model.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := models.Embed(t.Context(), model.RoleQuery, []string{"b"}); err != nil {
		t.Fatal(err)
	}

	if client.asked[0].Priority != connection.PriorityBackground || client.asked[1].Priority != connection.PriorityInteractive {
		t.Errorf("priorities = %v and %v", client.asked[0].Priority, client.asked[1].Priority)
	}
}

func TestTheProfileOfTheHostIsReadAsAConnectionProfile(t *testing.T) {
	client := &fakeClient{profile: connection.EmbeddingProfile{
		Model: "m", Source: "type https://example.org", Pooling: "cls", QueryPrefix: "q: ", DocumentPrefix: "d: ", Dimensions: 8, MaxTokens: 512,
	}}

	got, err := New(client).GetEmbeddingProfile(t.Context())

	want := model.ConnectionProfile{Model: "m", Source: "type https://example.org", Pooling: "cls", QueryPrefix: "q: ", DocumentPrefix: "d: ", Dimensions: 8, MaxTokens: 512}
	if err != nil || got != want {
		t.Errorf("profile = %+v, %v, want %+v", got, err, want)
	}
}

func TestFailuresOfTheConnectionsAreConnectionErrors(t *testing.T) {
	cases := map[string]error{
		"every row failed": &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "OpenRouter", Reason: "payment required"}}},
		"a row rejected":   &connection.RejectedError{ConnectionName: "Ollama", Reason: "key rejected"},
		"no row":           connection.ErrNoConnection,
	}
	for name, failure := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := New(&fakeClient{err: failure}).Embed(t.Context(), model.RoleDocument, []string{"a"})

			if failure, isConnection := errors.AsType[ports.ConnectionError](err); !isConnection || failure.Err == nil {
				t.Errorf("err = %v, want a connection error", err)
			}
		})
	}
}

func TestACancelledCallKeepsItsContextError(t *testing.T) {
	_, err := New(&fakeClient{err: context.Canceled}).Embed(t.Context(), model.RoleDocument, []string{"a"})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the cancellation", err)
	}
	if failure, isConnection := errors.AsType[ports.ConnectionError](err); isConnection && failure.Err != nil {
		t.Error("a cancellation was reported as a connection error")
	}
}
