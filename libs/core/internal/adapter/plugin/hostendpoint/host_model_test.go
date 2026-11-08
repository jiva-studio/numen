package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/usecase/modelcall"
	"github.com/jiva-studio/numen/libs/core/usecase/modelcall/modelcalltest"
)

// scriptedCalls is a ModelCalls that answers with what the test sets and records who asked.
type scriptedCalls struct {
	mu       sync.Mutex
	callers  []string
	complete connectionport.CompleteResult
	vectors  [][]float32
	profile  connectionport.EmbeddingProfile
	err      error
	request  connectionport.CompleteRequest
	texts    []string
	started  chan struct{}
}

func (s *scriptedCalls) record(ctx context.Context, caller string) error {
	s.mu.Lock()
	s.callers = append(s.callers, caller)
	started := s.started
	s.mu.Unlock()
	if started != nil {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}
	return s.err
}

func (s *scriptedCalls) Complete(ctx context.Context, caller string, request connectionport.CompleteRequest) (connectionport.CompleteResult, error) {
	s.request = request
	return s.complete, s.record(ctx, caller)
}

func (s *scriptedCalls) Embed(ctx context.Context, caller string, texts []string) ([][]float32, error) {
	s.texts = texts
	return s.vectors, s.record(ctx, caller)
}

func (s *scriptedCalls) GetEmbeddingProfile(ctx context.Context, caller string) (connectionport.EmbeddingProfile, error) {
	return s.profile, s.record(ctx, caller)
}

// recordingClient keeps every byte of every response the consumer receives.
type recordingClient struct {
	inner    connect.HTTPClient
	mu       sync.Mutex
	received strings.Builder
}

func (c *recordingClient) Do(request *http.Request) (*http.Response, error) {
	response, err := c.inner.Do(request)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	c.mu.Lock()
	fmt.Fprintf(&c.received, "%v %s", response.Header, body)
	c.mu.Unlock()
	response.Body = io.NopCloser(strings.NewReader(string(body)))
	return response, readErr
}

func newModelClient(t *testing.T, calls connectionport.ModelCalls, token string) (numenv1connect.HostModelServiceClient, *recordingClient) {
	t.Helper()
	fakeCallers := newCallers(nil)
	fakeCallers.modelCalls = calls
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostModelServiceHandler(&modelEndpoint{endpoint: &Endpoint{callers: fakeCallers}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	recording := &recordingClient{inner: server.Client()}
	return numenv1connect.NewHostModelServiceClient(recording, server.URL, connect.WithInterceptors(transport.AuthInterceptor(token))), recording
}

func TestCompleteIsAskedForThePluginTheTokenNames(t *testing.T) {
	calls := &scriptedCalls{complete: connectionport.CompleteResult{Text: "fixed"}}
	client, _ := newModelClient(t, calls, "token-b")
	temperature := 0.4

	response, err := client.Complete(t.Context(), connect.NewRequest(&v1.HostModelCompleteRequest{
		Messages: []*v1.HostModelMessage{{Role: "user", Text: "u"}}, Temperature: &temperature, MaxTokens: 33,
	}))

	if err != nil || response.Msg.GetText() != "fixed" || response.Msg.GetError() != v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("Complete = %v, %v", response, err)
	}
	if len(calls.callers) != 1 || calls.callers[0] != "md.numen.b" {
		t.Errorf("callers = %v, want md.numen.b: the plugin of the token, which holds no ai:complete permission", calls.callers)
	}
	if len(calls.request.Messages) != 1 || calls.request.Messages[0] != (connectionport.Message{Role: "user", Text: "u"}) ||
		*calls.request.Temperature != 0.4 || calls.request.MaxTokens != 33 {
		t.Errorf("request = %+v", calls.request)
	}
}

func TestACallWithoutAKnownTokenIsUnauthenticated(t *testing.T) {
	client, _ := newModelClient(t, &scriptedCalls{}, "no-such-token")
	_, err := client.Embed(t.Context(), connect.NewRequest(&v1.HostModelEmbedRequest{Texts: []string{"a"}}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestEmbedReturnsTheVectorsInOrder(t *testing.T) {
	calls := &scriptedCalls{vectors: [][]float32{{1, 2}, {3, 4}}}
	client, _ := newModelClient(t, calls, "token-a")

	response, err := client.Embed(t.Context(), connect.NewRequest(&v1.HostModelEmbedRequest{Texts: []string{"a", "b"}}))

	if err != nil || len(response.Msg.GetVectors()) != 2 || response.Msg.GetVectors()[1].GetValues()[0] != 3 {
		t.Errorf("Embed = %v, %v", response, err)
	}
	if len(calls.texts) != 2 {
		t.Errorf("texts = %v", calls.texts)
	}
}

func TestGetEmbeddingProfileReturnsTheProfile(t *testing.T) {
	calls := &scriptedCalls{profile: connectionport.EmbeddingProfile{
		Model: "te3", Source: "openrouter", Pooling: "mean", QueryPrefix: "q", DocumentPrefix: "d", Dimensions: 3, MaxTokens: 8000,
	}}
	client, _ := newModelClient(t, calls, "token-a")

	response, err := client.GetEmbeddingProfile(t.Context(), connect.NewRequest(&v1.HostModelGetEmbeddingProfileRequest{}))

	msg := response.Msg
	if err != nil || msg.GetModel() != "te3" || msg.GetSource() != "openrouter" || msg.GetPooling() != "mean" || msg.GetQueryPrefix() != "q" ||
		msg.GetDocumentPrefix() != "d" || msg.GetDimensions() != 3 || msg.GetMaxTokens() != 8000 {
		t.Errorf("GetEmbeddingProfile = %v, %v", response, err)
	}
}

func TestAHostWithNoModelCallsAnswersNoConnection(t *testing.T) {
	client, _ := newModelClient(t, nil, "token-a")
	response, err := client.Complete(t.Context(), connect.NewRequest(&v1.HostModelCompleteRequest{}))
	if err != nil || response.Msg.GetError() != v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION {
		t.Errorf("Complete = %v, %v, want a served NO_CONNECTION", response, err)
	}
}

func TestNoConnectionIsAServedErrorCode(t *testing.T) {
	client, _ := newModelClient(t, &scriptedCalls{err: connectionport.ErrNoConnection}, "token-a")
	response, err := client.Complete(t.Context(), connect.NewRequest(&v1.HostModelCompleteRequest{}))
	if err != nil || response.Msg.GetError() != v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION {
		t.Errorf("Complete = %v, %v", response, err)
	}
}

func TestAllFailedIsAServedErrorCodeThatNamesEachConnection(t *testing.T) {
	failure := &connectionport.AllFailedError{Failures: []connectionport.Failure{
		{ConnectionName: "OpenRouter", Reason: "payment required"}, {ConnectionName: "Ollama", Reason: "unreachable"},
	}}
	client, _ := newModelClient(t, &scriptedCalls{err: fmt.Errorf("embed: %w", failure)}, "token-a")

	response, err := client.Embed(t.Context(), connect.NewRequest(&v1.HostModelEmbedRequest{Texts: []string{"a"}}))

	if err != nil || response.Msg.GetError() != v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_ALL_FAILED {
		t.Fatalf("Embed = %v, %v", response, err)
	}
	failures := response.Msg.GetFailures()
	if len(failures) != 2 || failures[0].GetConnectionName() != "OpenRouter" || failures[0].GetReason() != "payment required" ||
		failures[1].GetConnectionName() != "Ollama" || failures[1].GetReason() != "unreachable" {
		t.Errorf("failures = %v", failures)
	}
	if response.Msg.GetDetail() != "OpenRouter: payment required; Ollama: unreachable" {
		t.Errorf("detail = %q", response.Msg.GetDetail())
	}
}

func TestARejectionIsAServedErrorCodeThatNamesTheConnection(t *testing.T) {
	client, _ := newModelClient(t, &scriptedCalls{err: &connectionport.RejectedError{ConnectionName: "OpenRouter", Reason: "bad request"}}, "token-a")

	response, err := client.GetEmbeddingProfile(t.Context(), connect.NewRequest(&v1.HostModelGetEmbeddingProfileRequest{}))

	if err != nil || response.Msg.GetError() != v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_REJECTED ||
		len(response.Msg.GetFailures()) != 1 || response.Msg.GetFailures()[0].GetConnectionName() != "OpenRouter" ||
		response.Msg.GetFailures()[0].GetReason() != "bad request" {
		t.Errorf("GetEmbeddingProfile = %v, %v", response, err)
	}
}

func TestAnyOtherFailureIsAStatusCode(t *testing.T) {
	client, _ := newModelClient(t, &scriptedCalls{err: errors.New("the settings are unreadable")}, "token-a")
	_, err := client.Complete(t.Context(), connect.NewRequest(&v1.HostModelCompleteRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("code = %v, want internal", connect.CodeOf(err))
	}
}

func TestACancelledCallReachesTheCallsAndReturnsCanceled(t *testing.T) {
	calls := &scriptedCalls{started: make(chan struct{})}
	client, _ := newModelClient(t, calls, "token-a")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		_, err := client.Complete(ctx, connect.NewRequest(&v1.HostModelCompleteRequest{}))
		done <- err
	}()
	<-calls.started

	cancel()

	if err := <-done; connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("code = %v, want canceled: %v", connect.CodeOf(err), err)
	}
}

func TestEveryErrorCodeOfTheSchemaIsProducedFromAFailure(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.HostModelErrorCode]error{
		v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION: connectionport.ErrNoConnection,
		v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_ALL_FAILED:    &connectionport.AllFailedError{},
		v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_REJECTED:      &connectionport.RejectedError{},
	}, func(failure error) v1.HostModelErrorCode {
		code, _ := getModelErrorCode(failure)
		return code
	})
}

// A consumer reaches the real calls through the endpoint and receives a key in nothing: not in an answer,
// not in the error of failed rows, not in a rejection, and not in the headers.
func TestAConsumerReceivesNoKeyInAnyAnswer(t *testing.T) {
	const keyOne, keyTwo = "sk-one-secret-0123", "sk-two-secret-4567"
	for name, arrange := range map[string]func(*modelcalltest.World){
		"an answer": func(*modelcalltest.World) {},
		"every row failed": func(w *modelcalltest.World) {
			w.Provider.FailAlways("https://one.test/v1", &connectionport.TryNextError{Reason: connectionport.FailureReasonKeyRejected})
			w.Provider.FailAlways("https://two.test/v1", &connectionport.TryNextError{Reason: connectionport.FailureReasonPaymentRequired})
		},
		"a rejection": func(w *modelcalltest.World) {
			w.Provider.FailAlways("https://one.test/v1", &connectionport.StopError{Reason: connectionport.FailureReasonBadRequest})
		},
		"a provider that echoes its key in an error": func(w *modelcalltest.World) {
			w.Provider.FailAlways("https://one.test/v1", errors.New("Bearer "+keyOne))
			w.Provider.FailAlways("https://two.test/v1", errors.New("Bearer "+keyTwo))
		},
	} {
		world := modelcalltest.New()
		one := world.AddServer(t, "One", "https://one.test/v1", keyOne)
		two := world.AddServer(t, "Two", "https://two.test/v1", keyTwo)
		world.Rows.Set("md.numen.b", "complete", modelcalltest.Row(one, "m"), modelcalltest.Row(two, "m"))
		world.Rows.Set("md.numen.b", "embed", modelcalltest.Row(one, "m"), modelcalltest.Row(two, "m"))
		arrange(world)
		client, recording := newModelClient(t, modelcall.NewCalls(world.Connections, world.Rows, world.Source, world.Clock, world.Notices), "token-b")

		_, _ = client.Complete(t.Context(), connect.NewRequest(&v1.HostModelCompleteRequest{}))
		_, _ = client.Embed(t.Context(), connect.NewRequest(&v1.HostModelEmbedRequest{Texts: []string{"a"}}))
		_, _ = client.GetEmbeddingProfile(t.Context(), connect.NewRequest(&v1.HostModelGetEmbeddingProfileRequest{}))

		if received := recording.received.String(); strings.Contains(received, "sk-") {
			t.Errorf("%s: the consumer received a key: %s", name, received)
		}
	}
}
