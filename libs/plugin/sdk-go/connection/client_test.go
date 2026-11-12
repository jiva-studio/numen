package connection

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/core/protocol/schematest"
)

type fakeHost struct {
	numenv1connect.UnimplementedHostModelServiceHandler
	token    string
	complete func(*v1.HostModelServiceCompleteRequest) *v1.HostModelServiceCompleteResponse
	embed    func(*v1.HostModelServiceEmbedRequest) *v1.HostModelServiceEmbedResponse
	profile  *v1.HostModelServiceGetEmbeddingProfileResponse
}

func (h *fakeHost) Complete(
	_ context.Context, request *connect.Request[v1.HostModelServiceCompleteRequest],
) (*connect.Response[v1.HostModelServiceCompleteResponse], error) {
	h.token = request.Header().Get("Authorization")
	return connect.NewResponse(h.complete(request.Msg)), nil
}

func (h *fakeHost) Embed(
	_ context.Context, request *connect.Request[v1.HostModelServiceEmbedRequest],
) (*connect.Response[v1.HostModelServiceEmbedResponse], error) {
	return connect.NewResponse(h.embed(request.Msg)), nil
}

func (h *fakeHost) GetEmbeddingProfile(
	context.Context, *connect.Request[v1.HostModelServiceGetEmbeddingProfileRequest],
) (*connect.Response[v1.HostModelServiceGetEmbeddingProfileResponse], error) {
	return connect.NewResponse(h.profile), nil
}

func newTestClient(t *testing.T, host *fakeHost) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostModelServiceHandler(host))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewClient(server.Client(), server.URL, "token-1")
}

func TestCompleteSendsMessagesAndTheTokenAndReturnsTheText(t *testing.T) {
	var received *v1.HostModelServiceCompleteRequest
	host := &fakeHost{complete: func(request *v1.HostModelServiceCompleteRequest) *v1.HostModelServiceCompleteResponse {
		received = request
		return &v1.HostModelServiceCompleteResponse{Text: "fixed"}
	}}
	temperature := 0.5
	result, err := newTestClient(t, host).Complete(t.Context(), CompleteRequest{
		Messages: []Message{{Role: "system", Text: "s"}, {Role: "user", Text: "u"}}, Temperature: &temperature, MaxTokens: 100,
	})
	if err != nil || result.Text != "fixed" {
		t.Fatalf("Complete = %v, %v", result, err)
	}
	if host.token != "Bearer token-1" {
		t.Errorf("Authorization = %q", host.token)
	}
	if len(received.GetMessages()) != 2 || received.GetMessages()[1].GetText() != "u" || received.GetTemperature() != 0.5 || received.GetMaxTokens() != 100 {
		t.Errorf("request = %v", received)
	}
}

func TestEmbedReturnsTheVectorsInOrder(t *testing.T) {
	host := &fakeHost{embed: func(request *v1.HostModelServiceEmbedRequest) *v1.HostModelServiceEmbedResponse {
		response := &v1.HostModelServiceEmbedResponse{}
		for i := range request.GetTexts() {
			response.Vectors = append(response.Vectors, &v1.FloatVector{Values: []float32{float32(i)}})
		}
		return response
	}}
	result, err := newTestClient(t, host).Embed(t.Context(), EmbedRequest{Texts: []string{"a", "b", "c"}})
	if err != nil || len(result.Vectors) != 3 || result.Vectors[2][0] != 2 {
		t.Fatalf("Embed = %v, %v", result, err)
	}
}

func TestGetEmbeddingProfileReturnsTheProfileOfTheRowThatAnswered(t *testing.T) {
	host := &fakeHost{profile: &v1.HostModelServiceGetEmbeddingProfileResponse{
		Model: "te3", Source: "openrouter", Pooling: "mean", QueryPrefix: "q", DocumentPrefix: "d", Dimensions: 3, MaxTokens: 8000,
	}}
	profile, err := newTestClient(t, host).GetEmbeddingProfile(t.Context())
	want := EmbeddingProfile{Model: "te3", Source: "openrouter", Pooling: "mean", QueryPrefix: "q", DocumentPrefix: "d", Dimensions: 3, MaxTokens: 8000}
	if err != nil || profile != want {
		t.Fatalf("GetEmbeddingProfile = %#v, %v, want %#v", profile, err, want)
	}
}

func TestANoConnectionResponseIsErrNoConnection(t *testing.T) {
	host := &fakeHost{complete: func(*v1.HostModelServiceCompleteRequest) *v1.HostModelServiceCompleteResponse {
		return &v1.HostModelServiceCompleteResponse{Error: v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION}
	}}
	_, err := newTestClient(t, host).Complete(t.Context(), CompleteRequest{})
	if !errors.Is(err, ErrNoConnection) {
		t.Errorf("err = %v, want ErrNoConnection", err)
	}
}

func TestAnAllFailedResponseNamesEveryConnectionAndReason(t *testing.T) {
	host := &fakeHost{embed: func(*v1.HostModelServiceEmbedRequest) *v1.HostModelServiceEmbedResponse {
		return &v1.HostModelServiceEmbedResponse{
			Error: v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_ALL_FAILED,
			Failures: []*v1.HostModelFailure{
				{ConnectionName: "OpenRouter", Reason: "payment required"}, {ConnectionName: "Ollama", Reason: "unreachable"},
			},
		}
	}}
	_, err := newTestClient(t, host).Embed(t.Context(), EmbedRequest{Texts: []string{"a"}})
	var failed *AllFailedError
	if !errors.As(err, &failed) {
		t.Fatalf("err = %v, want *AllFailedError", err)
	}
	if len(failed.Failures) != 2 || failed.Failures[0] != (Failure{ConnectionName: "OpenRouter", Reason: "payment required"}) ||
		failed.Failures[1] != (Failure{ConnectionName: "Ollama", Reason: "unreachable"}) {
		t.Errorf("failures = %v", failed.Failures)
	}
	if !strings.Contains(err.Error(), "OpenRouter: payment required; Ollama: unreachable") {
		t.Errorf("text = %q", err.Error())
	}
}

func TestARejectedResponseIsARejectedError(t *testing.T) {
	host := &fakeHost{complete: func(*v1.HostModelServiceCompleteRequest) *v1.HostModelServiceCompleteResponse {
		return &v1.HostModelServiceCompleteResponse{
			Error:    v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_REJECTED,
			Failures: []*v1.HostModelFailure{{ConnectionName: "OpenRouter", Reason: "bad request"}},
		}
	}}
	_, err := newTestClient(t, host).Complete(t.Context(), CompleteRequest{})
	var rejected *RejectedError
	if !errors.As(err, &rejected) || rejected.ConnectionName != "OpenRouter" || rejected.Reason != "bad request" {
		t.Errorf("err = %v, want a rejection by OpenRouter for a bad request", err)
	}
}

func TestEveryHostModelErrorCodeIsReadAsAnError(t *testing.T) {
	for code := range v1.HostModelErrorCode_name {
		if v1.HostModelErrorCode(code) == v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_UNSPECIFIED {
			continue
		}
		if readError(v1.HostModelErrorCode(code), "", nil) == nil {
			t.Errorf("code %v is read as success", v1.HostModelErrorCode(code))
		}
	}
}

func TestACanceledCallReturnsTheContextError(t *testing.T) {
	host := &fakeHost{complete: func(*v1.HostModelServiceCompleteRequest) *v1.HostModelServiceCompleteResponse { return nil }}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := newTestClient(t, host).Complete(ctx, CompleteRequest{}); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestNewClientFromVarsNeedsAHostPort(t *testing.T) {
	if _, err := NewClientFromVars(map[string]string{}); err == nil {
		t.Error("NewClientFromVars accepted variables with no host port")
	}
	if _, err := NewClientFromVars(map[string]string{"NUMEN_HOST_PORT": "4000", "NUMEN_PLUGIN_TOKEN": "t"}); err != nil {
		t.Errorf("NewClientFromVars = %v", err)
	}
}

func TestCompleteSendsThePriorityAndABackgroundOneWhereNoneIsSet(t *testing.T) {
	for name, want := range map[string]struct {
		priority Priority
		wire     v1.Priority
	}{
		"interactive": {PriorityInteractive, v1.Priority_PRIORITY_INTERACTIVE},
		"background":  {PriorityBackground, v1.Priority_PRIORITY_BACKGROUND},
		"unset":       {0, v1.Priority_PRIORITY_BACKGROUND},
	} {
		t.Run(name, func(t *testing.T) {
			var received v1.Priority
			host := &fakeHost{complete: func(request *v1.HostModelServiceCompleteRequest) *v1.HostModelServiceCompleteResponse {
				received = request.GetPriority()
				return &v1.HostModelServiceCompleteResponse{}
			}}

			_, err := newTestClient(t, host).Complete(t.Context(), CompleteRequest{Priority: want.priority})

			if err != nil || received != want.wire {
				t.Errorf("priority = %v, %v, want %v", received, err, want.wire)
			}
		})
	}
}

func TestEmbedSendsThePriority(t *testing.T) {
	var received v1.Priority
	host := &fakeHost{embed: func(request *v1.HostModelServiceEmbedRequest) *v1.HostModelServiceEmbedResponse {
		received = request.GetPriority()
		return &v1.HostModelServiceEmbedResponse{}
	}}

	_, err := newTestClient(t, host).Embed(t.Context(), EmbedRequest{Texts: []string{"a"}, Priority: PriorityInteractive})

	if err != nil || received != v1.Priority_PRIORITY_INTERACTIVE {
		t.Errorf("priority = %v, %v, want interactive", received, err)
	}
}

func TestEveryPriorityOfTheSchemaIsProducedFromAClientPriority(t *testing.T) {
	schematest.CheckProduced(t, map[v1.Priority]Priority{
		v1.Priority_PRIORITY_INTERACTIVE: PriorityInteractive,
		v1.Priority_PRIORITY_BACKGROUND:  PriorityBackground,
	}, writePriority)
}
