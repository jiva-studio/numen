package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	modelcalls "github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
)

// scriptedCalls is a ModelCalls that answers with what the test sets and records who asked.
type scriptedCalls struct {
	mu       sync.Mutex
	callers  []string
	complete modelcall.CompleteResult
	vectors  [][]float32
	profile  modelcall.EmbeddingProfile
	err      error
	request  modelcall.CompleteRequest
	texts    []string
	priority modelcall.Priority
	started  chan struct{}

	events      []modelcall.ChatEvent
	chatRequest modelcall.ChatRequest
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

func (s *scriptedCalls) Complete(ctx context.Context, caller string, request modelcall.CompleteRequest, priority modelcall.Priority) (modelcall.CompleteResult, error) {
	s.request, s.priority = request, priority
	return s.complete, s.record(ctx, caller)
}

func (s *scriptedCalls) Chat(
	ctx context.Context, caller string, request modelcall.ChatRequest, priority modelcall.Priority, send func(modelcall.ChatEvent) error,
) error {
	s.chatRequest, s.priority = request, priority
	for _, one := range s.events {
		if err := send(one); err != nil {
			return err
		}
	}
	return s.record(ctx, caller)
}

func (s *scriptedCalls) Embed(ctx context.Context, caller string, texts []string, priority modelcall.Priority) ([][]float32, error) {
	s.texts, s.priority = texts, priority
	return s.vectors, s.record(ctx, caller)
}

func (s *scriptedCalls) GetEmbeddingProfile(ctx context.Context, caller string) (modelcall.EmbeddingProfile, error) {
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
	readErr = errors.Join(readErr, response.Body.Close())
	c.mu.Lock()
	fmt.Fprintf(&c.received, "%v %s", response.Header, body)
	c.mu.Unlock()
	response.Body = io.NopCloser(strings.NewReader(string(body)))
	return response, readErr
}

func newModelClient(t *testing.T, calls ModelCalls, token string) (numenv1connect.HostModelServiceClient, *recordingClient) {
	t.Helper()
	fakeCallers := newCallers()
	fakeCallers.modelCalls = calls
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostModelServiceHandler(&modelEndpoint{endpoint: &Endpoint{callers: fakeCallers.asCallers()}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	recording := &recordingClient{inner: server.Client()}
	return numenv1connect.NewHostModelServiceClient(recording, server.URL, connect.WithInterceptors(transport.AuthInterceptor(token))), recording
}

func TestCompleteIsAskedForThePluginTheTokenNames(t *testing.T) {
	calls := &scriptedCalls{complete: modelcall.CompleteResult{Text: "fixed"}}
	client, _ := newModelClient(t, calls, "token-b")
	temperature := 0.4

	response, err := client.Complete(t.Context(), connect.NewRequest(&v1.HostModelServiceCompleteRequest{
		Messages: []*v1.HostModelMessage{{Role: "user", Text: "u"}}, Temperature: &temperature, MaxTokens: 33,
	}))

	if err != nil || response.Msg.GetText() != "fixed" || response.Msg.GetError() != v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("Complete = %v, %v", response, err)
	}
	if len(calls.callers) != 1 || calls.callers[0] != "md.numen.b" {
		t.Errorf("callers = %v, want md.numen.b: the plugin of the token", calls.callers)
	}
	if len(calls.request.Messages) != 1 || calls.request.Messages[0] != (modelcall.Message{Role: "user", Text: "u"}) ||
		*calls.request.Temperature != 0.4 || calls.request.MaxTokens != 33 {
		t.Errorf("request = %+v", calls.request)
	}
}

func TestACallWithoutAKnownTokenIsUnauthenticated(t *testing.T) {
	client, _ := newModelClient(t, &scriptedCalls{}, "no-such-token")
	_, err := client.Embed(t.Context(), connect.NewRequest(&v1.HostModelServiceEmbedRequest{Texts: []string{"a"}}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestEmbedReturnsTheVectorsInOrder(t *testing.T) {
	calls := &scriptedCalls{vectors: [][]float32{{1, 2}, {3, 4}}}
	client, _ := newModelClient(t, calls, "token-a")

	response, err := client.Embed(t.Context(), connect.NewRequest(&v1.HostModelServiceEmbedRequest{Texts: []string{"a", "b"}}))

	if err != nil || len(response.Msg.GetVectors()) != 2 || response.Msg.GetVectors()[1].GetValues()[0] != 3 {
		t.Errorf("Embed = %v, %v", response, err)
	}
	if len(calls.texts) != 2 {
		t.Errorf("texts = %v", calls.texts)
	}
}

func TestGetEmbeddingProfileReturnsTheProfile(t *testing.T) {
	calls := &scriptedCalls{profile: modelcall.EmbeddingProfile{
		Model: "te3", Source: "openrouter", Pooling: "mean", QueryPrefix: "q", DocumentPrefix: "d", Dimensions: 3, MaxTokens: 8000,
	}}
	client, _ := newModelClient(t, calls, "token-a")

	response, err := client.GetEmbeddingProfile(t.Context(), connect.NewRequest(&v1.HostModelServiceGetEmbeddingProfileRequest{}))

	msg := response.Msg
	if err != nil || msg.GetModel() != "te3" || msg.GetSource() != "openrouter" || msg.GetPooling() != "mean" || msg.GetQueryPrefix() != "q" ||
		msg.GetDocumentPrefix() != "d" || msg.GetDimensions() != 3 || msg.GetMaxTokens() != 8000 {
		t.Errorf("GetEmbeddingProfile = %v, %v", response, err)
	}
}

func TestAHostWithNoModelCallsAnswersNoConnection(t *testing.T) {
	client, _ := newModelClient(t, nil, "token-a")
	response, err := client.Complete(t.Context(), connect.NewRequest(&v1.HostModelServiceCompleteRequest{}))
	if err != nil || response.Msg.GetError() != v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION {
		t.Errorf("Complete = %v, %v, want a served NO_CONNECTION", response, err)
	}
}

func TestNoConnectionIsAServedErrorCode(t *testing.T) {
	client, _ := newModelClient(t, &scriptedCalls{err: connection.ErrNoConnection}, "token-a")
	response, err := client.Complete(t.Context(), connect.NewRequest(&v1.HostModelServiceCompleteRequest{}))
	if err != nil || response.Msg.GetError() != v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION {
		t.Errorf("Complete = %v, %v", response, err)
	}
}

func TestAllFailedIsAServedErrorCodeThatNamesEachConnection(t *testing.T) {
	failure := &connection.AllFailedError{Failures: []connection.Failure{
		{ConnectionName: "OpenRouter", Reason: "payment required"}, {ConnectionName: "Ollama", Reason: "unreachable"},
	}}
	client, _ := newModelClient(t, &scriptedCalls{err: fmt.Errorf("embed: %w", failure)}, "token-a")

	response, err := client.Embed(t.Context(), connect.NewRequest(&v1.HostModelServiceEmbedRequest{Texts: []string{"a"}}))

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
	client, _ := newModelClient(t, &scriptedCalls{err: &connection.RejectedError{ConnectionName: "OpenRouter", Reason: "bad request"}}, "token-a")

	response, err := client.GetEmbeddingProfile(t.Context(), connect.NewRequest(&v1.HostModelServiceGetEmbeddingProfileRequest{}))

	if err != nil || response.Msg.GetError() != v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_REJECTED ||
		len(response.Msg.GetFailures()) != 1 || response.Msg.GetFailures()[0].GetConnectionName() != "OpenRouter" ||
		response.Msg.GetFailures()[0].GetReason() != "bad request" {
		t.Errorf("GetEmbeddingProfile = %v, %v", response, err)
	}
}

func TestAnyOtherFailureIsAStatusCode(t *testing.T) {
	client, _ := newModelClient(t, &scriptedCalls{err: errors.New("the settings are unreadable")}, "token-a")
	_, err := client.Complete(t.Context(), connect.NewRequest(&v1.HostModelServiceCompleteRequest{}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("code = %v, want internal", connect.CodeOf(err))
	}
}

func TestACanceledCallReachesTheCallsAndReturnsCanceled(t *testing.T) {
	calls := &scriptedCalls{started: make(chan struct{})}
	client, _ := newModelClient(t, calls, "token-a")
	ctx, cancel := context.WithCancel(t.Context())
	outcome := make(chan error, 1)
	go func() {
		_, err := client.Complete(ctx, connect.NewRequest(&v1.HostModelServiceCompleteRequest{}))
		outcome <- err
	}()
	<-calls.started

	cancel()

	if err := <-outcome; connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("code = %v, want canceled: %v", connect.CodeOf(err), err)
	}
}

func TestEveryErrorCodeOfTheSchemaIsProducedFromAFailure(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.HostModelErrorCode]error{
		v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION: connection.ErrNoConnection,
		v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_ALL_FAILED:    &connection.AllFailedError{},
		v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_REJECTED:      &connection.RejectedError{},
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
			w.Provider.FailAlways("https://one.test/v1", &connection.TryNextError{Reason: connection.FailureReasonKeyRejected})
			w.Provider.FailAlways("https://two.test/v1", &connection.TryNextError{Reason: connection.FailureReasonPaymentRequired})
		},
		"a rejection": func(w *modelcalltest.World) {
			w.Provider.FailAlways("https://one.test/v1", &connection.StopError{Reason: connection.FailureReasonBadRequest})
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
		client, recording := newModelClient(t, modelcalls.NewCalls(world.Resolve, world.ReadSecrets, world.Rows, world.Source, world.Clock, world.Waits.Wait, world.Notices), "token-b")

		_, completeErr := client.Complete(t.Context(), connect.NewRequest(&v1.HostModelServiceCompleteRequest{}))
		_, embedErr := client.Embed(t.Context(), connect.NewRequest(&v1.HostModelServiceEmbedRequest{Texts: []string{"a"}}))
		_, profileErr := client.GetEmbeddingProfile(t.Context(), connect.NewRequest(&v1.HostModelServiceGetEmbeddingProfileRequest{}))
		t.Logf("calls: %v; %v; %v", completeErr, embedErr, profileErr)

		if received := recording.received.String(); strings.Contains(received, "sk-") {
			t.Errorf("%s: the consumer received a key: %s", name, received)
		}
	}
}

func TestCompleteCarriesThePriorityOfTheRequest(t *testing.T) {
	for name, want := range map[string]struct {
		wire v1.Priority
		port modelcall.Priority
	}{
		"interactive": {v1.Priority_PRIORITY_INTERACTIVE, modelcall.PriorityInteractive},
		"background":  {v1.Priority_PRIORITY_BACKGROUND, modelcall.PriorityBackground},
		"unspecified": {v1.Priority_PRIORITY_UNSPECIFIED, modelcall.PriorityBackground},
	} {
		t.Run(name, func(t *testing.T) {
			calls := &scriptedCalls{}
			client, _ := newModelClient(t, calls, "token-a")

			_, err := client.Complete(t.Context(), connect.NewRequest(&v1.HostModelServiceCompleteRequest{Priority: want.wire}))

			if err != nil || calls.priority != want.port {
				t.Errorf("priority = %v, %v, want %v", calls.priority, err, want.port)
			}
		})
	}
}

func TestEmbedCarriesThePriorityOfTheRequest(t *testing.T) {
	calls := &scriptedCalls{}
	client, _ := newModelClient(t, calls, "token-a")

	_, err := client.Embed(t.Context(), connect.NewRequest(&v1.HostModelServiceEmbedRequest{
		Texts: []string{"a"}, Priority: v1.Priority_PRIORITY_INTERACTIVE,
	}))

	if err != nil || calls.priority != modelcall.PriorityInteractive {
		t.Errorf("priority = %v, %v, want interactive", calls.priority, err)
	}
}

func TestThePluginEndpointDoesNotMountModelService(t *testing.T) {
	fakeCallers := newCallers()
	endpoint, err := Open(t.Context(), "127.0.0.1:0", fakeCallers.asCallers())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	client := numenv1connect.NewModelServiceClient(http.DefaultClient, fmt.Sprintf("http://127.0.0.1:%d", endpoint.GetPort()),
		connect.WithInterceptors(transport.AuthInterceptor("token-a")))

	_, callErr := client.Complete(t.Context(), connect.NewRequest(&v1.ModelServiceCompleteRequest{PluginId: "md.numen.b"}))

	if connect.CodeOf(callErr) != connect.CodeUnimplemented && connect.CodeOf(callErr) != connect.CodeNotFound {
		t.Errorf("code = %v, want the service to be absent", connect.CodeOf(callErr))
	}
}

func TestEveryPriorityOfTheSchemaIsReadAsThePortOneIt(t *testing.T) {
	want := map[v1.Priority]modelcall.Priority{
		v1.Priority_PRIORITY_INTERACTIVE: modelcall.PriorityInteractive,
		v1.Priority_PRIORITY_BACKGROUND:  modelcall.PriorityBackground,
	}
	testsupport.CheckHandled(t, func(priority v1.Priority) bool { return readPriority(priority) == want[priority] })
}
