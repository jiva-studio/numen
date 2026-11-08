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

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const (
	pluginID = "md.example.provider"
	token    = "plugin-token"
)

// scriptedService is a ProviderService whose answers the test sets.
type scriptedService struct {
	numenv1connect.UnimplementedProviderServiceHandler
	mu        sync.Mutex
	complete  func(*v1.ProviderCompleteRequest) (*v1.ProviderCompleteResponse, error)
	authority []string
	received  []*v1.ProviderCall
}

func (s *scriptedService) Complete(
	_ context.Context, request *connect.Request[v1.ProviderCompleteRequest],
) (*connect.Response[v1.ProviderCompleteResponse], error) {
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
	_ context.Context, request *connect.Request[v1.ProviderEmbedRequest],
) (*connect.Response[v1.ProviderEmbedResponse], error) {
	response := &v1.ProviderEmbedResponse{}
	for i := range request.Msg.GetTexts() {
		response.Vectors = append(response.Vectors, &v1.FloatVector{Values: []float32{float32(i), 2}})
	}
	return connect.NewResponse(response), nil
}

func (s *scriptedService) ListModels(
	context.Context, *connect.Request[v1.ProviderListModelsRequest],
) (*connect.Response[v1.ProviderListModelsResponse], error) {
	return connect.NewResponse(&v1.ProviderListModelsResponse{Models: []*v1.ProviderModel{
		{Id: "m1", Title: "One", Operations: []string{"embed"}}, {Id: "m2"},
	}}), nil
}

func (s *scriptedService) ListReasoningLevels(
	context.Context, *connect.Request[v1.ProviderListReasoningLevelsRequest],
) (*connect.Response[v1.ProviderListReasoningLevelsResponse], error) {
	return connect.NewResponse(&v1.ProviderListReasoningLevelsResponse{Levels: []*v1.ProviderReasoningLevel{{Value: "low", Title: "Low"}}}), nil
}

func (s *scriptedService) GetModelProfile(
	context.Context, *connect.Request[v1.ProviderGetModelProfileRequest],
) (*connect.Response[v1.ProviderGetModelProfileResponse], error) {
	return connect.NewResponse(&v1.ProviderGetModelProfileResponse{
		IsKnown: true, Dimensions: 3, MaxTokens: 512, Pooling: "mean", QueryPrefix: "q", DocumentPrefix: "d", Source: "openrouter",
	}), nil
}

func (s *scriptedService) ProbeEmbeddingWidth(
	context.Context, *connect.Request[v1.ProviderProbeEmbeddingWidthRequest],
) (*connect.Response[v1.ProviderProbeEmbeddingWidthResponse], error) {
	return connect.NewResponse(&v1.ProviderProbeEmbeddingWidthResponse{Dimensions: 9}), nil
}

// activator hands out the in-process endpoint of the service and counts the releases.
type activator struct {
	mu       sync.Mutex
	endpoint pluginport.Endpoint
	err      error
	released []string
}

func (a *activator) EnsureActive(context.Context, string) (pluginport.Endpoint, error) {
	return a.endpoint, a.err
}

func (a *activator) Release(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.released = append(a.released, id)
}

func newBridge(t *testing.T, service numenv1connect.ProviderServiceHandler) (connectionport.Provider, func(), *activator) {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewProviderServiceHandler(service))
	host := &activator{endpoint: pluginport.Endpoint{Host: "inprocess", Token: token, Transport: plugin.NewInProcessTransport(mux)}}
	provider, release, err := providerbridge.NewSource(host).OpenProvider(t.Context(), pluginID)
	if err != nil {
		t.Fatal(err)
	}
	return provider, release, host
}

func answerWith(response *v1.ProviderCompleteResponse) func(*v1.ProviderCompleteRequest) (*v1.ProviderCompleteResponse, error) {
	return func(*v1.ProviderCompleteRequest) (*v1.ProviderCompleteResponse, error) { return response, nil }
}

func TestCompleteCarriesTheCallAndTheRequestAndReturnsTheText(t *testing.T) {
	temperature := 0.5
	var seen *v1.ProviderCompleteRequest
	service := &scriptedService{complete: func(request *v1.ProviderCompleteRequest) (*v1.ProviderCompleteResponse, error) {
		seen = request
		return &v1.ProviderCompleteResponse{Text: "answer"}, nil
	}}
	provider, _, _ := newBridge(t, service)

	result, err := provider.Complete(t.Context(), connectionport.ProviderCall{
		Type: "server", Fields: map[string]string{"baseUrl": "https://x"}, Secrets: map[string]string{"apiKey": "sk-1"}, Model: "m", Reasoning: "low",
	}, connectionport.CompleteRequest{
		Messages: []connectionport.Message{{Role: "user", Text: "hi"}}, Temperature: &temperature, MaxTokens: 10,
	})

	if err != nil || result.Text != "answer" {
		t.Fatalf("Complete = %v, %v", result, err)
	}
	call := seen.GetCall()
	if call.GetType() != "server" || call.GetModel() != "m" || call.GetReasoning() != "low" || call.GetFields()["baseUrl"] != "https://x" ||
		len(call.GetSecrets()) != 1 || call.GetSecrets()[0].GetKey() != "apiKey" || call.GetSecrets()[0].GetValue() != "sk-1" {
		t.Errorf("call = %v", call)
	}
	if seen.GetTemperature() != 0.5 || seen.GetMaxTokens() != 10 || len(seen.GetMessages()) != 1 || seen.GetMessages()[0].GetText() != "hi" {
		t.Errorf("request = %v", seen)
	}
	if service.authority[0] != "Bearer "+token {
		t.Errorf("Authorization = %q, want the token of the plugin", service.authority[0])
	}
}

func TestATryNextFailureBecomesATryNextError(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{complete: answerWith(&v1.ProviderCompleteResponse{Failure: &v1.ProviderFailure{
		Kind: v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_TRY_NEXT, Reason: v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED, RetryAfterMs: 2500,
	}})})

	_, err := provider.Complete(t.Context(), connectionport.ProviderCall{}, connectionport.CompleteRequest{})

	var tryNext *connectionport.TryNextError
	if !errors.As(err, &tryNext) || tryNext.Reason != connectionport.FailureReasonRateLimited || tryNext.RetryAfter != 2500*time.Millisecond {
		t.Errorf("err = %v, want a try-next error for a rate limit after 2.5 s", err)
	}
}

func TestAStopFailureBecomesAStopError(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{complete: answerWith(&v1.ProviderCompleteResponse{Failure: &v1.ProviderFailure{
		Kind: v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_STOP, Reason: v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_BAD_REQUEST,
	}})})

	_, err := provider.Complete(t.Context(), connectionport.ProviderCall{}, connectionport.CompleteRequest{})

	var stopped *connectionport.StopError
	if !errors.As(err, &stopped) || stopped.Reason != connectionport.FailureReasonBadRequest {
		t.Errorf("err = %v, want a stop error for a bad request", err)
	}
}

func TestAFailureOfNoKindIsAStopOfUnknownReason(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{complete: answerWith(&v1.ProviderCompleteResponse{Failure: &v1.ProviderFailure{}})})

	_, err := provider.Complete(t.Context(), connectionport.ProviderCall{}, connectionport.CompleteRequest{})

	var stopped *connectionport.StopError
	if !errors.As(err, &stopped) || stopped.Reason != connectionport.FailureReasonUnknown {
		t.Errorf("err = %v, want a stop error of unknown reason", err)
	}
}

func TestARoleThePluginDoesNotServeIsAStopOfUnknownReason(t *testing.T) {
	provider, _, _ := newBridge(t, numenv1connect.UnimplementedProviderServiceHandler{})

	_, err := provider.ProbeEmbeddingWidth(t.Context(), connectionport.ProviderCall{})

	var stopped *connectionport.StopError
	if !errors.As(err, &stopped) || stopped.Reason != connectionport.FailureReasonUnknown {
		t.Errorf("err = %v, want a stop error of unknown reason", err)
	}
}

func TestAServiceThatFailsWithAStatusCodeIsAnErrorOfNoFailureKind(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{complete: func(*v1.ProviderCompleteRequest) (*v1.ProviderCompleteResponse, error) {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("connection reset"))
	}})

	_, err := provider.Complete(t.Context(), connectionport.ProviderCall{}, connectionport.CompleteRequest{})

	var tryNext *connectionport.TryNextError
	var stopped *connectionport.StopError
	if err == nil || errors.As(err, &tryNext) || errors.As(err, &stopped) {
		t.Errorf("err = %v, want a plain error the call path treats as unreachable", err)
	}
}

func TestACancelledCallReturnsTheErrorOfTheContext(t *testing.T) {
	started := make(chan struct{})
	provider, _, _ := newBridge(t, &scriptedService{complete: func(*v1.ProviderCompleteRequest) (*v1.ProviderCompleteResponse, error) {
		close(started)
		<-time.After(10 * time.Second)
		return &v1.ProviderCompleteResponse{}, nil
	}})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		_, err := provider.Complete(ctx, connectionport.ProviderCall{}, connectionport.CompleteRequest{})
		done <- err
	}()
	<-started

	cancel()

	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestEmbedReturnsTheVectorsInOrder(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{})

	vectors, err := provider.Embed(t.Context(), connectionport.ProviderCall{}, []string{"a", "b"})

	if err != nil || len(vectors) != 2 || vectors[1][0] != 1 || vectors[1][1] != 2 {
		t.Errorf("Embed = %v, %v", vectors, err)
	}
}

func TestTheCatalogAndTheProfileAreCarried(t *testing.T) {
	provider, _, _ := newBridge(t, &scriptedService{})

	models, err := provider.ListModels(t.Context(), connectionport.ProviderCall{}, "embed")
	if err != nil || len(models) != 2 || models[0].ID != "m1" || models[0].Title != "One" || !slices.Equal(models[0].Operations, []string{"embed"}) || models[1].Operations != nil {
		t.Errorf("ListModels = %v, %v", models, err)
	}
	levels, err := provider.ListReasoningLevels(t.Context(), connectionport.ProviderCall{})
	if err != nil || len(levels) != 1 || levels[0] != (connectionport.ReasoningLevel{Value: "low", Title: "Low"}) {
		t.Errorf("ListReasoningLevels = %v, %v", levels, err)
	}
	profile, err := provider.GetModelProfile(t.Context(), connectionport.ProviderCall{})
	want := connectionport.ModelProfile{IsKnown: true, Dimensions: 3, MaxTokens: 512, Pooling: "mean", QueryPrefix: "q", DocumentPrefix: "d", Source: "openrouter"}
	if err != nil || profile != want {
		t.Errorf("GetModelProfile = %+v, %v, want %+v", profile, err, want)
	}
	width, err := provider.ProbeEmbeddingWidth(t.Context(), connectionport.ProviderCall{})
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
