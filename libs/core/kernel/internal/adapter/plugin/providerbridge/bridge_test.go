package providerbridge_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

const (
	pluginID = "md.example.provider"
	token    = "plugin-token"
)

// scriptedService is a ProviderService that sends the frames the test sets and records what it received.
type scriptedService struct {
	numenv1connect.UnimplementedProviderServiceHandler
	mu        sync.Mutex
	frames    []*v1.ProviderServiceCallResponse
	err       error
	block     bool
	started   chan struct{}
	ended     chan error
	authority []string
	received  []*v1.ProviderServiceCallRequest
}

func (s *scriptedService) Call(
	ctx context.Context, request *connect.Request[v1.ProviderServiceCallRequest], stream *connect.ServerStream[v1.ProviderServiceCallResponse],
) error {
	s.mu.Lock()
	s.authority = append(s.authority, request.Header().Get("Authorization"))
	s.received = append(s.received, request.Msg)
	s.mu.Unlock()
	for _, one := range s.frames {
		if err := stream.Send(one); err != nil {
			return err
		}
	}
	if s.block {
		close(s.started)
		<-ctx.Done()
		s.ended <- ctx.Err()
		return ctx.Err()
	}
	return s.err
}

func data(text string) *v1.ProviderServiceCallResponse {
	return &v1.ProviderServiceCallResponse{Frame: &v1.ProviderServiceCallResponse_Data{Data: []byte(text)}}
}

func done(payload []byte) *v1.ProviderServiceCallResponse {
	return &v1.ProviderServiceCallResponse{Frame: &v1.ProviderServiceCallResponse_Done{Done: payload}}
}

func failure(kind v1.FailureKind, reason v1.FailureReason, retryAfterMs int64) *v1.ProviderServiceCallResponse {
	return &v1.ProviderServiceCallResponse{Frame: &v1.ProviderServiceCallResponse_Failure{Failure: &v1.ProviderFailure{
		Kind: kind, Reason: reason, RetryAfterMs: retryAfterMs,
	}}}
}

// activator hands out the in-process endpoint of the service and counts the releases.
type activator struct {
	mu       sync.Mutex
	endpoint transport.Endpoint
	err      error
	client   *http.Client
	released []string
}

func (a *activator) EnsureActive(context.Context, string) (transport.Endpoint, error) {
	return a.endpoint, a.err
}

// GetClient returns the in-process client of the service, or the default client where the service listens on the loopback.
func (a *activator) GetClient(transport.Endpoint) *http.Client {
	if a.client == nil {
		return http.DefaultClient
	}
	return a.client
}

func (a *activator) Release(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.released = append(a.released, id)
}

func newActivator(t *testing.T, service numenv1connect.ProviderServiceHandler) *activator {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewProviderServiceHandler(service))
	return &activator{
		endpoint: transport.Endpoint{Host: "inprocess", Token: token},
		client:   &http.Client{Transport: transport.NewInProcessTransport(mux)},
	}
}

func newBridge(t *testing.T, service numenv1connect.ProviderServiceHandler) (connectionport.Provider, func(), *activator) {
	t.Helper()
	host := newActivator(t, service)
	provider, release, err := providerbridge.NewSource(host).OpenProvider(t.Context(), pluginID)
	if err != nil {
		t.Fatal(err)
	}
	return provider, release, host
}

func callCollecting(ctx context.Context, provider connectionport.Provider, frames *[]string) ([]byte, error) {
	return provider.Call(ctx, "llm.chat", modelcall.ProviderCall{}, []byte("turn"), func(data []byte) error {
		*frames = append(*frames, string(data))
		return nil
	})
}

func TestCallCarriesTheOperationTheConnectionAndThePayloadAndStreamsTheFramesInOrder(t *testing.T) {
	service := &scriptedService{frames: []*v1.ProviderServiceCallResponse{data("Let me "), data("look."), done([]byte("finish"))}}
	provider, _, _ := newBridge(t, service)
	call := modelcall.ProviderCall{
		ConnectionID: "c0000beef", Type: "openrouter", Fields: map[string]string{"baseUrl": "https://openrouter.test/api/v1"},
		Secrets: map[string]string{"apiKey": "sk-or-123"}, Options: map[string]string{"model": "google/gemini-2.5-flash", "reasoning": "low"},
	}
	var frames []string

	got, err := provider.Call(t.Context(), "llm.chat", call, []byte("turn"), func(data []byte) error {
		frames = append(frames, string(data))
		return nil
	})

	if err != nil || string(got) != "finish" || !slices.Equal(frames, []string{"Let me ", "look."}) {
		t.Fatalf("Call = %q, %v, frames %v", got, err, frames)
	}
	received := service.received[0]
	wire := received.GetCall()
	if received.GetOperation() != "llm.chat" || string(received.GetPayload()) != "turn" ||
		wire.GetConnectionId() != "c0000beef" || wire.GetType() != "openrouter" || wire.GetFields()["baseUrl"] != "https://openrouter.test/api/v1" ||
		wire.GetOptions()["model"] != "google/gemini-2.5-flash" || wire.GetOptions()["reasoning"] != "low" ||
		len(wire.GetSecrets()) != 1 || wire.GetSecrets()[0].GetKey() != "apiKey" || wire.GetSecrets()[0].GetValue() != "sk-or-123" {
		t.Errorf("received = %+v", received)
	}
	if service.authority[0] != "Bearer "+token {
		t.Errorf("Authorization = %q", service.authority[0])
	}
}

func TestATryNextFailureBecomesATryNextError(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{frames: []*v1.ProviderServiceCallResponse{
		failure(v1.FailureKind_FAILURE_KIND_TRY_NEXT, v1.FailureReason_FAILURE_REASON_RATE_LIMITED, 30000),
	}})
	var frames []string

	_, err := callCollecting(t.Context(), provider, &frames)

	var tryNext *connection.TryNextError
	if !errors.As(err, &tryNext) || tryNext.Reason != connection.FailureReasonRateLimited || tryNext.RetryAfter != 30*time.Second {
		t.Errorf("err = %v, want a try-next error of rate limited after 30s", err)
	}
}

func TestAFailureAfterTheFirstFrameKeepsTheFramesAndBecomesAStopError(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{frames: []*v1.ProviderServiceCallResponse{
		data("partial"), failure(v1.FailureKind_FAILURE_KIND_STOP, v1.FailureReason_FAILURE_REASON_INVALID_REQUEST, 0),
	}})
	var frames []string

	_, err := callCollecting(t.Context(), provider, &frames)

	var stopError *connection.StopError
	if !errors.As(err, &stopError) || stopError.Reason != connection.FailureReasonInvalidRequest || !slices.Equal(frames, []string{"partial"}) {
		t.Errorf("err = %v, frames = %v", err, frames)
	}
}

func TestAStreamThatEndsWithNeitherADoneNorAFailureIsAnErrorOfNoFailureKind(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{frames: []*v1.ProviderServiceCallResponse{data("partial")}})
	var frames []string

	_, err := callCollecting(t.Context(), provider, &frames)

	var tryNext *connection.TryNextError
	var stopError *connection.StopError
	if err == nil || errors.As(err, &tryNext) || errors.As(err, &stopError) {
		t.Errorf("err = %v, want an error of no failure kind", err)
	}
}

func TestAnOperationThePluginDoesNotServeIsAStopOfAnInvalidRequest(t *testing.T) {
	provider, _, _ := newBridge(t, numenv1connect.UnimplementedProviderServiceHandler{})
	var frames []string

	_, err := callCollecting(t.Context(), provider, &frames)

	var stopError *connection.StopError
	if !errors.As(err, &stopError) || stopError.Reason != connection.FailureReasonInvalidRequest {
		t.Errorf("err = %v, want a stop of an invalid request", err)
	}
}

func TestAServiceThatFailsWithAStatusCodeIsAnErrorOfNoFailureKind(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{err: connect.NewError(connect.CodeUnavailable, errors.New("the plugin restarted"))})
	var frames []string

	_, err := callCollecting(t.Context(), provider, &frames)

	var tryNext *connection.TryNextError
	var stopError *connection.StopError
	if err == nil || errors.As(err, &tryNext) || errors.As(err, &stopError) {
		t.Errorf("err = %v, want an error of no failure kind", err)
	}
}

func TestAnErrorOfTheReceiverEndsTheCallAndIsReturnedAsItIs(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{frames: []*v1.ProviderServiceCallResponse{data("a"), data("b"), done(nil)}})
	consumerFailure := errors.New("the consumer went away")

	_, err := provider.Call(t.Context(), "llm.chat", modelcall.ProviderCall{}, nil, func([]byte) error { return consumerFailure })

	if !errors.Is(err, consumerFailure) {
		t.Errorf("err = %v, want the error of the receiver", err)
	}
}

func TestListOptionsSendsTheOperationsOfTheSettingAndReadsTheOptionsOfTheDoneFrame(t *testing.T) {
	listed, err := proto.Marshal(&v1.Options{Options: []*v1.Option{{Value: "m1", Title: "One", Group: "Free"}, {Value: "m2"}}})
	if err != nil {
		t.Fatal(err)
	}
	service := &scriptedService{frames: []*v1.ProviderServiceCallResponse{done(listed)}}
	provider, _, _ := newBridge(t, service)

	options, err := provider.ListOptions(t.Context(), "llm.models", modelcall.ProviderCall{Options: map[string]string{"reasoning": "low"}}, []string{"llm.embed"})

	want := []modelcall.Option{{Value: "m1", Title: "One", Group: "Free"}, {Value: "m2"}}
	if err != nil || !slices.Equal(options, want) {
		t.Fatalf("options = %v, %v, want %v", options, err, want)
	}
	var request v1.OptionsRequest
	if err := proto.Unmarshal(service.received[0].GetPayload(), &request); err != nil || !slices.Equal(request.GetOperations(), []string{"llm.embed"}) {
		t.Errorf("payload = %v, %v", request.GetOperations(), err)
	}
	if service.received[0].GetOperation() != "llm.models" || service.received[0].GetCall().GetOptions()["reasoning"] != "low" {
		t.Errorf("received = %+v", service.received[0])
	}
}

func TestADoneFrameThatHoldsNoOptionsIsAStopOfAnInvalidRequest(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{frames: []*v1.ProviderServiceCallResponse{done([]byte{0xff, 0xff})}})

	_, err := provider.ListOptions(t.Context(), "llm.models", modelcall.ProviderCall{}, nil)

	var stopError *connection.StopError
	if !errors.As(err, &stopError) || stopError.Reason != connection.FailureReasonInvalidRequest {
		t.Errorf("err = %v, want a stop of an invalid request", err)
	}
}

func TestTheLeaseIsReleasedWhenTheCallerReleasesTheProvider(t *testing.T) {
	_, release, host := newBridge(t, &scriptedService{})
	if len(host.released) != 0 {
		t.Fatalf("released before the caller did: %v", host.released)
	}

	release()
	release()

	if !slices.Equal(host.released, []string{pluginID}) {
		t.Errorf("released = %v, want the plugin once", host.released)
	}
}

func TestAPluginThatCannotBeStartedIsAnError(t *testing.T) {
	crash := errors.New("exit status 1")

	_, _, err := providerbridge.NewSource(&activator{err: crash}).OpenProvider(t.Context(), pluginID)

	if !errors.Is(err, crash) {
		t.Errorf("err = %v, want the failure to start", err)
	}
}
