package hostmodel

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

type askedCall struct {
	operation string
	priority  connection.Priority
}

// fakeCaller answers each operation with the done payload set for it, or fails every call with err.
type fakeCaller struct {
	done  map[string]proto.Message
	err   error
	asked []askedCall
}

func (f *fakeCaller) Call(_ context.Context, operation string, _ []byte, priority connection.Priority) (connection.Frames, error) {
	f.asked = append(f.asked, askedCall{operation: operation, priority: priority})
	if f.err != nil {
		return nil, f.err
	}
	done, err := proto.Marshal(f.done[operation])
	if err != nil {
		return nil, err
	}
	return &doneFrames{done: done}, nil
}

// doneFrames is a call with no data frame that ends with done.
type doneFrames struct{ done []byte }

func (d *doneFrames) Next() bool   { return false }
func (d *doneFrames) Data() []byte { return nil }
func (d *doneFrames) Done() []byte { return d.done }
func (d *doneFrames) Err() error   { return nil }
func (d *doneFrames) Close() error { return nil }

func newCaller() *fakeCaller {
	return &fakeCaller{done: map[string]proto.Message{
		llm.OperationEmbed: &llm.EmbedResult{Vectors: []*llm.Vector{{Values: []float32{1}}}},
	}}
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
	caller := &fakeCaller{done: map[string]proto.Message{llm.OperationProfile: &llm.Profile{
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

func TestACanceledCallKeepsItsContextError(t *testing.T) {
	_, err := New(&fakeCaller{err: context.Canceled}).Embed(t.Context(), model.RoleDocument, []string{"a"})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the cancellation", err)
	}
	if failure, isConnection := errors.AsType[ports.ConnectionError](err); isConnection && failure.Err != nil {
		t.Error("a cancellation was reported as a connection error")
	}
}
