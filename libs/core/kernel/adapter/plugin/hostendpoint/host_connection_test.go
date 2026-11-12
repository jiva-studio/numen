package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	modelcalls "github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
)

// scriptedCalls is a ModelCalls that sends the frames the test sets, answers done, and records who asked.
type scriptedCalls struct {
	mu      sync.Mutex
	callers []string
	frames  []string
	done    string
	err     error
	request modelcall.Request
	started chan struct{}
}

func (s *scriptedCalls) Call(ctx context.Context, caller string, request modelcall.Request, send modelcall.Send) ([]byte, error) {
	s.mu.Lock()
	s.callers = append(s.callers, caller)
	s.request = request
	started := s.started
	s.mu.Unlock()
	for _, one := range s.frames {
		if err := send([]byte(one)); err != nil {
			return nil, err
		}
	}
	if started != nil {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if s.err != nil {
		return nil, s.err
	}
	return []byte(s.done), nil
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

func newConnectionClient(t *testing.T, calls ModelCalls, token string) (numenv1connect.HostConnectionServiceClient, *recordingClient) {
	t.Helper()
	fakeCallers := newCallers()
	fakeCallers.modelCalls = calls
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostConnectionServiceHandler(&connectionEndpoint{endpoint: &Endpoint{callers: fakeCallers.asCallers()}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	recording := &recordingClient{inner: server.Client()}
	return numenv1connect.NewHostConnectionServiceClient(recording, server.URL, connect.WithInterceptors(transport.AuthInterceptor(token))), recording
}

// received is what a consumer read from a call: the data frames, the done payload and the failure.
type received struct {
	frames  []string
	done    []byte
	isDone  bool
	failure *v1.CallFailure
}

func call(ctx context.Context, client numenv1connect.HostConnectionServiceClient, request *v1.HostConnectionServiceCallRequest) (received, error) {
	stream, err := client.Call(ctx, connect.NewRequest(request))
	if err != nil {
		return received{}, err
	}
	defer func() { _ = stream.Close() }()
	var got received
	for stream.Receive() {
		switch frame := stream.Msg().GetFrame().(type) {
		case *v1.HostConnectionServiceCallResponse_Data:
			got.frames = append(got.frames, string(frame.Data))
		case *v1.HostConnectionServiceCallResponse_Done:
			got.done, got.isDone = frame.Done, true
		case *v1.HostConnectionServiceCallResponse_Failure:
			got.failure = frame.Failure
		}
	}
	return got, stream.Err()
}

func TestACallIsCarriedOutForThePluginTheTokenNamesAndStreamsItsFrames(t *testing.T) {
	calls := &scriptedCalls{frames: []string{"Let me ", "look."}, done: "finish"}
	client, _ := newConnectionClient(t, calls, "token-b")

	got, err := call(t.Context(), client, &v1.HostConnectionServiceCallRequest{Operation: "llm.chat", Payload: []byte("list the notes")})

	if err != nil || !slices.Equal(got.frames, []string{"Let me ", "look."}) || string(got.done) != "finish" || got.failure != nil {
		t.Fatalf("Call = %+v, %v", got, err)
	}
	if !slices.Equal(calls.callers, []string{"md.numen.b"}) || calls.request.Operation != "llm.chat" || string(calls.request.Payload) != "list the notes" {
		t.Errorf("callers = %v, request = %+v", calls.callers, calls.request)
	}
}

func TestACallWithoutAKnownTokenIsUnauthenticated(t *testing.T) {
	calls := &scriptedCalls{}
	client, _ := newConnectionClient(t, calls, "token-unknown")

	_, err := call(t.Context(), client, &v1.HostConnectionServiceCallRequest{Operation: "llm.complete"})

	if connect.CodeOf(err) != connect.CodeUnauthenticated || len(calls.callers) != 0 {
		t.Errorf("code = %v, callers = %v", connect.CodeOf(err), calls.callers)
	}
}

func TestAHostWithNoModelCallsAnswersNoConnection(t *testing.T) {
	client, _ := newConnectionClient(t, nil, "token-a")

	got, err := call(t.Context(), client, &v1.HostConnectionServiceCallRequest{Operation: "llm.complete"})

	if err != nil || got.failure.GetError() != v1.CallErrorCode_CALL_ERROR_CODE_NO_CONNECTION {
		t.Errorf("Call = %+v, %v, want a served NO_CONNECTION", got, err)
	}
}

func TestAllFailedIsAServedFailureThatNamesEachConnection(t *testing.T) {
	failure := &connection.AllFailedError{Failures: []connection.Failure{
		{ConnectionName: "OpenRouter", Reason: "payment required"}, {ConnectionName: "Ollama", Reason: "unavailable"},
	}}
	client, _ := newConnectionClient(t, &scriptedCalls{err: fmt.Errorf("embed: %w", failure)}, "token-a")

	got, err := call(t.Context(), client, &v1.HostConnectionServiceCallRequest{Operation: "llm.embed"})

	if err != nil || got.failure.GetError() != v1.CallErrorCode_CALL_ERROR_CODE_ALL_FAILED {
		t.Fatalf("Call = %+v, %v", got, err)
	}
	failures := got.failure.GetFailures()
	if len(failures) != 2 || failures[0].GetConnectionName() != "OpenRouter" || failures[0].GetReason() != "payment required" ||
		failures[1].GetConnectionName() != "Ollama" || failures[1].GetReason() != "unavailable" {
		t.Errorf("failures = %v", failures)
	}
	if got.failure.GetDetail() != "OpenRouter: payment required; Ollama: unavailable" {
		t.Errorf("detail = %q", got.failure.GetDetail())
	}
}

func TestAFailureAfterTheFirstFrameKeepsTheFramesAndEndsTheStreamWithTheFailure(t *testing.T) {
	failure := &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "OpenRouter", Reason: "rate limited"}}}
	client, _ := newConnectionClient(t, &scriptedCalls{frames: []string{"partial"}, err: failure}, "token-a")

	got, err := call(t.Context(), client, &v1.HostConnectionServiceCallRequest{Operation: "llm.chat"})

	if err != nil || !slices.Equal(got.frames, []string{"partial"}) || got.isDone ||
		got.failure.GetError() != v1.CallErrorCode_CALL_ERROR_CODE_ALL_FAILED {
		t.Errorf("Call = %+v, %v", got, err)
	}
}

func TestAStopIsAServedFailureThatNamesTheConnection(t *testing.T) {
	client, _ := newConnectionClient(t, &scriptedCalls{err: &connection.StoppedError{ConnectionName: "OpenRouter", Reason: "invalid request"}}, "token-a")

	got, err := call(t.Context(), client, &v1.HostConnectionServiceCallRequest{Operation: "llm.profile"})

	failures := got.failure.GetFailures()
	if err != nil || got.failure.GetError() != v1.CallErrorCode_CALL_ERROR_CODE_STOPPED ||
		len(failures) != 1 || failures[0].GetConnectionName() != "OpenRouter" || failures[0].GetReason() != "invalid request" {
		t.Errorf("Call = %+v, %v", got, err)
	}
}

func TestAnyOtherFailureIsAStatusCode(t *testing.T) {
	client, _ := newConnectionClient(t, &scriptedCalls{err: errors.New("the settings are unreadable")}, "token-a")

	_, err := call(t.Context(), client, &v1.HostConnectionServiceCallRequest{Operation: "llm.complete"})

	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("code = %v, want internal", connect.CodeOf(err))
	}
}

func TestACanceledCallReachesTheCallsAndReturnsCanceled(t *testing.T) {
	calls := &scriptedCalls{started: make(chan struct{})}
	client, _ := newConnectionClient(t, calls, "token-a")
	ctx, cancel := context.WithCancel(t.Context())
	outcome := make(chan error, 1)
	go func() {
		_, err := call(ctx, client, &v1.HostConnectionServiceCallRequest{Operation: "llm.chat"})
		outcome <- err
	}()
	<-calls.started

	cancel()

	if err := <-outcome; connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("code = %v, want canceled: %v", connect.CodeOf(err), err)
	}
}

func TestEveryErrorCodeOfTheSchemaIsProducedFromAFailure(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.CallErrorCode]error{
		v1.CallErrorCode_CALL_ERROR_CODE_NO_CONNECTION: connection.ErrNoConnection,
		v1.CallErrorCode_CALL_ERROR_CODE_ALL_FAILED:    &connection.AllFailedError{},
		v1.CallErrorCode_CALL_ERROR_CODE_STOPPED:       &connection.StoppedError{},
	}, func(failure error) v1.CallErrorCode {
		served, _ := writeCallFailure(context.Background(), failure)
		return served.GetError()
	})
}

// A consumer reaches the real calls through the endpoint and receives a key in nothing: not in an answer,
// not in the error of failed rows, not in a stop, and not in the headers.
func TestAConsumerReceivesNoKeyInAnyAnswer(t *testing.T) {
	const keyOne, keyTwo = "sk-one-secret-0123", "sk-two-secret-4567"
	for name, arrange := range map[string]func(*modelcalltest.World){
		"an answer": func(*modelcalltest.World) {},
		"every row failed": func(w *modelcalltest.World) {
			w.Provider.FailAlways("https://one.test/v1", &connection.TryNextError{Reason: connection.FailureReasonRejectedKey})
			w.Provider.FailAlways("https://two.test/v1", &connection.TryNextError{Reason: connection.FailureReasonPaymentRequired})
		},
		"a stop": func(w *modelcalltest.World) {
			w.Provider.FailAlways("https://one.test/v1", &connection.StopError{Reason: connection.FailureReasonInvalidRequest})
		},
		"a provider that echoes its key in an error": func(w *modelcalltest.World) {
			w.Provider.FailAlways("https://one.test/v1", errors.New("Bearer "+keyOne))
			w.Provider.FailAlways("https://two.test/v1", errors.New("Bearer "+keyTwo))
		},
		"a provider that echoes its key after a frame": func(w *modelcalltest.World) {
			w.Provider.Script("https://one.test/v1", errors.New("Bearer "+keyOne), "partial")
		},
	} {
		world := modelcalltest.New()
		one := world.AddServer(t, "One", "https://one.test/v1", keyOne)
		two := world.AddServer(t, "Two", "https://two.test/v1", keyTwo)
		world.Rows.Set("md.numen.b", modelcalltest.CompleteOperation, modelcalltest.Row(one, "m"), modelcalltest.Row(two, "m"))
		world.Rows.Set("md.numen.b", modelcalltest.ChatOperation, modelcalltest.Row(one, "m"), modelcalltest.Row(two, "m"))
		arrange(world)
		client, recording := newConnectionClient(t, modelcalls.NewCalls(lookupsOf(world), world.Source, world.Clock, world.Waits.Wait, world.Notices.ReportSetAside), "token-b")

		_, completeErr := call(t.Context(), client, &v1.HostConnectionServiceCallRequest{Operation: modelcalltest.CompleteOperation})
		_, chatErr := call(t.Context(), client, &v1.HostConnectionServiceCallRequest{Operation: modelcalltest.ChatOperation})
		t.Logf("calls: %v; %v", completeErr, chatErr)

		if text := recording.received.String(); strings.Contains(text, "sk-") {
			t.Errorf("%s: the consumer received a key: %s", name, text)
		}
	}
}

func TestACallCarriesThePriorityOfTheRequest(t *testing.T) {
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
			client, _ := newConnectionClient(t, calls, "token-a")

			_, err := call(t.Context(), client, &v1.HostConnectionServiceCallRequest{Operation: "llm.embed", Priority: want.wire})

			if err != nil || calls.request.Priority != want.port {
				t.Errorf("priority = %v, %v, want %v", calls.request.Priority, err, want.port)
			}
		})
	}
}

func TestThePluginEndpointDoesNotMountConnectionCallService(t *testing.T) {
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
	client := numenv1connect.NewConnectionCallServiceClient(http.DefaultClient, fmt.Sprintf("http://127.0.0.1:%d", endpoint.GetPort()),
		connect.WithInterceptors(transport.AuthInterceptor("token-a")))

	stream, callErr := client.Call(t.Context(), connect.NewRequest(&v1.ConnectionCallServiceCallRequest{PluginId: "md.numen.b", Operation: "llm.complete"}))
	if callErr == nil {
		for stream.Receive() {
		}
		callErr = stream.Err()
	}

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

func lookupsOf(world *modelcalltest.World) modelcalls.Connections {
	return modelcalls.Connections{Resolve: world.Resolve, ReadSecrets: world.ReadSecrets, Rows: world.Rows}
}
