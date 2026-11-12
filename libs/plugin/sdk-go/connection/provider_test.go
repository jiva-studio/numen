package connection

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

var errPlain = errors.New("HTTP 500 from the service: Authorization: Bearer sk-live-123")

type scriptedCompleter struct {
	err      error
	received Call
	request  ProviderCompleteRequest
}

func (c *scriptedCompleter) Complete(_ context.Context, call Call, request ProviderCompleteRequest) (CompleteResult, error) {
	c.received, c.request = call, request
	if c.err != nil {
		return CompleteResult{}, c.err
	}
	return CompleteResult{Text: "answer"}, nil
}

func newProviderClient(t *testing.T, provider Provider) numenv1connect.ProviderServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(NewHandler(provider))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewProviderServiceClient(server.Client(), server.URL)
}

func newCompleteRequest() *connect.Request[v1.ProviderServiceCompleteRequest] {
	temperature := 0.25
	return connect.NewRequest(&v1.ProviderServiceCompleteRequest{
		Call: &v1.ProviderCall{
			Type: "openrouter", Model: "m", Reasoning: "low", Fields: map[string]string{"baseUrl": "https://x"},
			Secrets: []*v1.ProviderSecret{{Key: "apiKey", Value: "sk-1"}},
		},
		Messages:    []*v1.ProviderMessage{{Role: "user", Text: "hi"}},
		Temperature: &temperature,
		MaxTokens:   64,
	})
}

func TestACompleteCallReachesTheProviderWithItsSnapshotAndRequest(t *testing.T) {
	completer := &scriptedCompleter{}
	client := newProviderClient(t, Provider{Completer: completer})
	response, err := client.Complete(t.Context(), newCompleteRequest())
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetText() != "answer" || response.Msg.GetFailure() != nil {
		t.Errorf("response = %v, want the text and no failure", response.Msg)
	}
	if completer.received.Type != "openrouter" || completer.received.Model != "m" || completer.received.Reasoning != "low" ||
		completer.received.Secrets["apiKey"].Value != "sk-1" || completer.received.Fields["baseUrl"] != "https://x" {
		t.Errorf("call = %#v", completer.received)
	}
	if completer.request.MaxTokens != 64 || completer.request.Temperature == nil || *completer.request.Temperature != 0.25 ||
		len(completer.request.Messages) != 1 || completer.request.Messages[0] != (Message{Role: "user", Text: "hi"}) {
		t.Errorf("request = %#v", completer.request)
	}
}

func TestATryNextFailureIsAServedFailureNotAStatusCode(t *testing.T) {
	client := newProviderClient(t, Provider{Completer: &scriptedCompleter{err: NewTryNextError(ReasonRateLimited, 7*time.Second)}})
	response, err := client.Complete(t.Context(), newCompleteRequest())
	if err != nil {
		t.Fatalf("a failure came as a status code: %v", err)
	}
	failure := response.Msg.GetFailure()
	if failure.GetKind() != v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_TRY_NEXT ||
		failure.GetReason() != v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED || failure.GetRetryAfterMs() != 7000 {
		t.Errorf("failure = %v", failure)
	}
	if response.Msg.GetText() != "" {
		t.Errorf("text = %q on a failed call", response.Msg.GetText())
	}
}

func TestAnErrorOfNoKindIsAStopWhoseTextDoesNotLeave(t *testing.T) {
	client := newProviderClient(t, Provider{Completer: &scriptedCompleter{err: errPlain}})
	response, err := client.Complete(t.Context(), newCompleteRequest())
	if err != nil {
		t.Fatal(err)
	}
	failure := response.Msg.GetFailure()
	if failure.GetKind() != v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_STOP ||
		failure.GetReason() != v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNKNOWN {
		t.Errorf("failure = %v, want a stop of unknown reason", failure)
	}
	if response.Msg.GetText() != "" || failure.GetRetryAfterMs() != 0 {
		t.Errorf("response carries more than the failure: %v", response.Msg)
	}
}

func TestACancelledCallReturnsTheStatusCodeNotAFailure(t *testing.T) {
	client := newProviderClient(t, Provider{Completer: &scriptedCompleter{err: context.Canceled}})
	_, err := client.Complete(t.Context(), newCompleteRequest())
	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("code = %v, want canceled: %v", connect.CodeOf(err), err)
	}
}

func TestARoleTheProviderLacksIsUnimplemented(t *testing.T) {
	client := newProviderClient(t, Provider{Completer: &scriptedCompleter{}})
	_, err := client.Embed(t.Context(), connect.NewRequest(&v1.ProviderServiceEmbedRequest{Call: &v1.ProviderCall{}, Texts: []string{"a"}}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("code = %v, want unimplemented: %v", connect.CodeOf(err), err)
	}
}

type fakeProviderRoles struct{}

func (fakeProviderRoles) Embed(_ context.Context, _ Call, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = []float32{float32(i), 1}
	}
	return vectors, nil
}

func (fakeProviderRoles) ListModels(_ context.Context, _ Call, operation string) ([]Model, error) {
	return []Model{{ID: "m1", Title: "Model one", Operations: []string{operation}}, {ID: "m2"}}, nil
}

func (fakeProviderRoles) ListReasoningLevels(context.Context, Call) ([]ReasoningLevel, error) {
	return []ReasoningLevel{{Value: "low", Title: "Low"}}, nil
}

func (fakeProviderRoles) GetModelProfile(context.Context, Call) (ModelProfile, error) {
	return ModelProfile{IsKnown: true, Dimensions: 3, MaxTokens: 8191, Pooling: "mean", QueryPrefix: "q: ", DocumentPrefix: "d: ", Source: "openrouter"}, nil
}

func (fakeProviderRoles) ProbeEmbeddingWidth(context.Context, Call) (int, error) { return 5, nil }

func TestEveryRoleIsServedOverTheWire(t *testing.T) {
	roles := fakeProviderRoles{}
	client := newProviderClient(t, Provider{Embedder: roles, Models: roles, Profiles: roles})
	call := &v1.ProviderCall{Model: "m"}

	embedded, err := client.Embed(t.Context(), connect.NewRequest(&v1.ProviderServiceEmbedRequest{Call: call, Texts: []string{"a", "b"}}))
	if err != nil || len(embedded.Msg.GetVectors()) != 2 || embedded.Msg.GetVectors()[1].GetValues()[0] != 1 {
		t.Errorf("Embed = %v, %v", embedded, err)
	}
	models, err := client.ListModels(t.Context(), connect.NewRequest(&v1.ProviderServiceListModelsRequest{Call: call, Operation: "embed"}))
	if err != nil || len(models.Msg.GetModels()) != 2 || models.Msg.GetModels()[0].GetOperations()[0] != "embed" {
		t.Errorf("ListModels = %v, %v", models, err)
	}
	levels, err := client.ListReasoningLevels(t.Context(), connect.NewRequest(&v1.ProviderServiceListReasoningLevelsRequest{Call: call}))
	if err != nil || len(levels.Msg.GetLevels()) != 1 || levels.Msg.GetLevels()[0].GetValue() != "low" {
		t.Errorf("ListReasoningLevels = %v, %v", levels, err)
	}
	profile, err := client.GetModelProfile(t.Context(), connect.NewRequest(&v1.ProviderServiceGetModelProfileRequest{Call: call}))
	if err != nil || !profile.Msg.GetIsKnown() || profile.Msg.GetDimensions() != 3 || profile.Msg.GetMaxTokens() != 8191 ||
		profile.Msg.GetPooling() != "mean" || profile.Msg.GetQueryPrefix() != "q: " || profile.Msg.GetDocumentPrefix() != "d: " ||
		profile.Msg.GetSource() != "openrouter" {
		t.Errorf("GetModelProfile = %v, %v", profile, err)
	}
	probe, err := client.ProbeEmbeddingWidth(t.Context(), connect.NewRequest(&v1.ProviderServiceProbeEmbeddingWidthRequest{Call: call}))
	if err != nil || probe.Msg.GetDimensions() != 5 {
		t.Errorf("ProbeEmbeddingWidth = %v, %v", probe, err)
	}
}

func TestTheReservedJobCallsAreUnimplemented(t *testing.T) {
	client := newProviderClient(t, Provider{Completer: &scriptedCompleter{}})
	_, err := client.SubmitJob(t.Context(), connect.NewRequest(&v1.ProviderServiceSubmitJobRequest{Call: &v1.ProviderCall{}}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("SubmitJob code = %v, want unimplemented", connect.CodeOf(err))
	}
}
