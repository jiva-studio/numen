package providerbridge_test

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

const (
	pluginID = "md.example.provider"
	token    = "plugin-token"
)

// scriptedService is a ProviderService whose answers the test sets.
type scriptedService struct {
	numenv1connect.UnimplementedProviderServiceHandler
	mu        sync.Mutex
	complete  func(*v1.ProviderServiceCompleteRequest) (*v1.ProviderServiceCompleteResponse, error)
	authority []string
	received  []*v1.ProviderCall
}

func (s *scriptedService) Complete(
	_ context.Context, request *connect.Request[v1.ProviderServiceCompleteRequest],
) (*connect.Response[v1.ProviderServiceCompleteResponse], error) {
	s.mu.Lock()
	s.authority = append(s.authority, request.Header().Get("Authorization"))
	s.received = append(s.received, request.Msg.GetCall())
	s.mu.Unlock()
	response, err := s.complete(request.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (s *scriptedService) Embed(
	_ context.Context, request *connect.Request[v1.ProviderServiceEmbedRequest],
) (*connect.Response[v1.ProviderServiceEmbedResponse], error) {
	response := &v1.ProviderServiceEmbedResponse{}
	for i := range request.Msg.GetTexts() {
		response.Vectors = append(response.Vectors, &v1.FloatVector{Values: []float32{float32(i), 2}})
	}
	return connect.NewResponse(response), nil
}

func (s *scriptedService) ListModels(
	context.Context, *connect.Request[v1.ProviderServiceListModelsRequest],
) (*connect.Response[v1.ProviderServiceListModelsResponse], error) {
	return connect.NewResponse(&v1.ProviderServiceListModelsResponse{Models: []*v1.ProviderModel{
		{Id: "m1", Title: "One", Operations: []string{"embed"}}, {Id: "m2"},
	}}), nil
}

func (s *scriptedService) ListReasoningLevels(
	context.Context, *connect.Request[v1.ProviderServiceListReasoningLevelsRequest],
) (*connect.Response[v1.ProviderServiceListReasoningLevelsResponse], error) {
	return connect.NewResponse(&v1.ProviderServiceListReasoningLevelsResponse{Levels: []*v1.ProviderReasoningLevel{{Value: "low", Title: "Low"}}}), nil
}

func (s *scriptedService) GetModelProfile(
	context.Context, *connect.Request[v1.ProviderServiceGetModelProfileRequest],
) (*connect.Response[v1.ProviderServiceGetModelProfileResponse], error) {
	return connect.NewResponse(&v1.ProviderServiceGetModelProfileResponse{
		IsKnown: true, Dimensions: 3, MaxTokens: 512, Pooling: "mean", QueryPrefix: "q", DocumentPrefix: "d", Source: "openrouter",
	}), nil
}

func (s *scriptedService) ProbeEmbeddingWidth(
	context.Context, *connect.Request[v1.ProviderServiceProbeEmbeddingWidthRequest],
) (*connect.Response[v1.ProviderServiceProbeEmbeddingWidthResponse], error) {
	return connect.NewResponse(&v1.ProviderServiceProbeEmbeddingWidthResponse{Dimensions: 9}), nil
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

func newCatalog(t *testing.T, service numenv1connect.ProviderServiceHandler) connectionport.ModelCatalog {
	t.Helper()
	host := newActivator(t, service)
	catalog, _, err := providerbridge.NewSource(host).OpenCatalog(t.Context(), pluginID)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func answerWith(response *v1.ProviderServiceCompleteResponse) func(*v1.ProviderServiceCompleteRequest) (*v1.ProviderServiceCompleteResponse, error) {
	return func(*v1.ProviderServiceCompleteRequest) (*v1.ProviderServiceCompleteResponse, error) {
		return response, nil
	}
}

func TestCompleteCarriesTheCallAndTheRequestAndReturnsTheText(t *testing.T) {
	temperature := 0.5
	var received *v1.ProviderServiceCompleteRequest
	service := &scriptedService{complete: func(request *v1.ProviderServiceCompleteRequest) (*v1.ProviderServiceCompleteResponse, error) {
		received = request
		return &v1.ProviderServiceCompleteResponse{Text: "answer"}, nil
	}}
	provider, _, _ := newBridge(t, service)

	result, err := provider.Complete(t.Context(), modelcall.ProviderCall{
		Type: "server", Fields: map[string]string{"baseUrl": "https://x"}, Secrets: map[string]string{"apiKey": "sk-1"}, Model: "m", Reasoning: "low",
	}, modelcall.CompleteRequest{
		Messages: []modelcall.Message{{Role: "user", Text: "hi"}}, Temperature: &temperature, MaxTokens: 10,
	})

	if err != nil || result.Text != "answer" {
		t.Fatalf("Complete = %v, %v", result, err)
	}
	call := received.GetCall()
	if call.GetType() != "server" || call.GetModel() != "m" || call.GetReasoning() != "low" || call.GetFields()["baseUrl"] != "https://x" ||
		len(call.GetSecrets()) != 1 || call.GetSecrets()[0].GetKey() != "apiKey" || call.GetSecrets()[0].GetValue() != "sk-1" {
		t.Errorf("call = %v", call)
	}
	if received.GetTemperature() != 0.5 || received.GetMaxTokens() != 10 || len(received.GetMessages()) != 1 || received.GetMessages()[0].GetText() != "hi" {
		t.Errorf("request = %v", received)
	}
	if service.authority[0] != "Bearer "+token {
		t.Errorf("Authorization = %q, want the token of the plugin", service.authority[0])
	}
}

func TestATryNextFailureBecomesATryNextError(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{complete: answerWith(&v1.ProviderServiceCompleteResponse{Failure: &v1.ProviderFailure{
		Kind: v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_TRY_NEXT, Reason: v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED, RetryAfterMs: 2500,
	}})})

	_, err := provider.Complete(t.Context(), modelcall.ProviderCall{}, modelcall.CompleteRequest{})

	var tryNext *connection.TryNextError
	if !errors.As(err, &tryNext) || tryNext.Reason != connection.FailureReasonRateLimited || tryNext.RetryAfter != 2500*time.Millisecond {
		t.Errorf("err = %v, want a try-next error for a rate limit after 2.5 s", err)
	}
}

func TestAStopFailureBecomesAStopError(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{complete: answerWith(&v1.ProviderServiceCompleteResponse{Failure: &v1.ProviderFailure{
		Kind: v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_STOP, Reason: v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_BAD_REQUEST,
	}})})

	_, err := provider.Complete(t.Context(), modelcall.ProviderCall{}, modelcall.CompleteRequest{})

	var stopError *connection.StopError
	if !errors.As(err, &stopError) || stopError.Reason != connection.FailureReasonBadRequest {
		t.Errorf("err = %v, want a stop error for a bad request", err)
	}
}

func TestAFailureOfNoKindIsAStopOfUnknownReason(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{complete: answerWith(&v1.ProviderServiceCompleteResponse{Failure: &v1.ProviderFailure{}})})

	_, err := provider.Complete(t.Context(), modelcall.ProviderCall{}, modelcall.CompleteRequest{})

	var stopError *connection.StopError
	if !errors.As(err, &stopError) || stopError.Reason != connection.FailureReasonUnknown {
		t.Errorf("err = %v, want a stop error of unknown reason", err)
	}
}

func TestARoleThePluginDoesNotServeIsAStopOfUnknownReason(t *testing.T) {
	provider, _, _ := newBridge(t, numenv1connect.UnimplementedProviderServiceHandler{})

	_, err := provider.ProbeEmbeddingWidth(t.Context(), modelcall.ProviderCall{})

	var stopError *connection.StopError
	if !errors.As(err, &stopError) || stopError.Reason != connection.FailureReasonUnknown {
		t.Errorf("err = %v, want a stop error of unknown reason", err)
	}
}

func TestAServiceThatFailsWithAStatusCodeIsAnErrorOfNoFailureKind(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{complete: func(*v1.ProviderServiceCompleteRequest) (*v1.ProviderServiceCompleteResponse, error) {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("connection reset"))
	}})

	_, err := provider.Complete(t.Context(), modelcall.ProviderCall{}, modelcall.CompleteRequest{})

	var tryNext *connection.TryNextError
	var stopError *connection.StopError
	if err == nil || errors.As(err, &tryNext) || errors.As(err, &stopError) {
		t.Errorf("err = %v, want a plain error the call path treats as unreachable", err)
	}
}

func TestACanceledCallReturnsTheErrorOfTheContext(t *testing.T) {
	started := make(chan struct{})
	provider, _, _ := newBridge(t, &scriptedService{complete: func(*v1.ProviderServiceCompleteRequest) (*v1.ProviderServiceCompleteResponse, error) {
		close(started)
		<-time.After(10 * time.Second)
		return &v1.ProviderServiceCompleteResponse{}, nil
	}})
	ctx, cancel := context.WithCancel(t.Context())
	outcome := make(chan error, 1)
	go func() {
		_, err := provider.Complete(ctx, modelcall.ProviderCall{}, modelcall.CompleteRequest{})
		outcome <- err
	}()
	<-started

	cancel()

	if err := <-outcome; !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestEmbedReturnsTheVectorsInOrder(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{})

	vectors, err := provider.Embed(t.Context(), modelcall.ProviderCall{}, []string{"a", "b"})

	if err != nil || len(vectors) != 2 || vectors[1][0] != 1 || vectors[1][1] != 2 {
		t.Errorf("Embed = %v, %v", vectors, err)
	}
}

func TestTheCatalogAndTheProfileAreCarried(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{})
	catalog := newCatalog(t, &scriptedService{})

	models, err := catalog.ListModels(t.Context(), modelcall.ProviderCall{}, "embed")
	if err != nil || len(models) != 2 || models[0].ID != "m1" || models[0].Title != "One" || !slices.Equal(models[0].Operations, []string{"embed"}) || models[1].Operations != nil {
		t.Errorf("ListModels = %v, %v", models, err)
	}
	levels, err := catalog.ListReasoningLevels(t.Context(), modelcall.ProviderCall{})
	if err != nil || len(levels) != 1 || levels[0] != (modelcall.ReasoningLevel{Value: "low", Title: "Low"}) {
		t.Errorf("ListReasoningLevels = %v, %v", levels, err)
	}
	profile, err := provider.GetModelProfile(t.Context(), modelcall.ProviderCall{})
	want := modelcall.ModelProfile{IsKnown: true, Dimensions: 3, MaxTokens: 512, Pooling: "mean", QueryPrefix: "q", DocumentPrefix: "d", Source: "openrouter"}
	if err != nil || profile != want {
		t.Errorf("GetModelProfile = %+v, %v, want %+v", profile, err, want)
	}
	width, err := provider.ProbeEmbeddingWidth(t.Context(), modelcall.ProviderCall{})
	if err != nil || width != 9 {
		t.Errorf("ProbeEmbeddingWidth = %d, %v", width, err)
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
