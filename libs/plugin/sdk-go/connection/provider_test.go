package connection

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/core/protocol/schematest"
)

var errPlain = errors.New("HTTP 500 from the service: Authorization: Bearer sk-live-123")

func newProviderClient(t *testing.T, provider Provider) numenv1connect.ProviderServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(NewHandler(provider))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewProviderServiceClient(server.Client(), server.URL)
}

func newCallRequest(operation string) *connect.Request[v1.ProviderServiceCallRequest] {
	return connect.NewRequest(&v1.ProviderServiceCallRequest{
		Operation: operation,
		Call: &v1.ProviderCall{
			ConnectionId: "c1f0a9e20", Type: "openrouter", Fields: map[string]string{"baseUrl": "https://openrouter.test/api/v1"},
			Secrets: []*v1.ProviderSecret{{Key: "apiKey", Value: "sk-or-1"}}, Options: map[string]string{"model": "google/gemini-2.5-flash"},
		},
		Payload: []byte("list the notes"),
	})
}

// served is what the core read from a call to a provider.
type served struct {
	frames  []string
	done    string
	isDone  bool
	failure *v1.ProviderFailure
}

func readCallResponse(t *testing.T, client numenv1connect.ProviderServiceClient, request *connect.Request[v1.ProviderServiceCallRequest]) (served, error) {
	t.Helper()
	stream, err := client.Call(t.Context(), request)
	if err != nil {
		return served{}, err
	}
	var got served
	for stream.Receive() {
		switch frame := stream.Msg().GetFrame().(type) {
		case *v1.ProviderServiceCallResponse_Data:
			got.frames = append(got.frames, string(frame.Data))
		case *v1.ProviderServiceCallResponse_Done:
			got.done, got.isDone = string(frame.Done), true
		case *v1.ProviderServiceCallResponse_Failure:
			got.failure = frame.Failure
		}
	}
	return got, stream.Err()
}

func TestAnOperationReceivesTheConnectionAndThePayloadAndItsFramesReachTheCore(t *testing.T) {
	var received Call
	var payload string
	client := newProviderClient(t, Provider{"llm.chat": func(_ context.Context, call Call, request []byte, send func([]byte) error) ([]byte, error) {
		received, payload = call, string(request)
		if err := send([]byte("Let me ")); err != nil {
			return nil, err
		}
		if err := send([]byte("look.")); err != nil {
			return nil, err
		}
		return []byte("finish"), nil
	}})

	got, err := readCallResponse(t, client, newCallRequest("llm.chat"))

	if err != nil || !slices.Equal(got.frames, []string{"Let me ", "look."}) || got.done != "finish" || got.failure != nil {
		t.Fatalf("Call = %+v, %v", got, err)
	}
	if received.ConnectionID != "c1f0a9e20" || received.Type != "openrouter" || received.Fields["baseUrl"] != "https://openrouter.test/api/v1" ||
		received.Secrets["apiKey"].Value != "sk-or-1" || received.Options["model"] != "google/gemini-2.5-flash" || payload != "list the notes" {
		t.Errorf("call = %+v, payload = %q", received, payload)
	}
}

func TestAnEmptyDoneIsADoneFrame(t *testing.T) {
	client := newProviderClient(t, Provider{"llm.complete": func(context.Context, Call, []byte, func([]byte) error) ([]byte, error) {
		return nil, nil
	}})

	got, err := readCallResponse(t, client, newCallRequest("llm.complete"))

	if err != nil || !got.isDone || got.done != "" {
		t.Errorf("Call = %+v, %v, want an empty done", got, err)
	}
}

func TestAFailureOfAnOperationIsAFailureFrameOfItsKindAndReason(t *testing.T) {
	for name, c := range map[string]struct {
		err  error
		want *v1.ProviderFailure
	}{
		"a rate limit": {
			NewTryNextError(ReasonRateLimited, 1500*time.Millisecond),
			&v1.ProviderFailure{Kind: v1.FailureKind_FAILURE_KIND_TRY_NEXT, Reason: v1.FailureReason_FAILURE_REASON_RATE_LIMITED, RetryAfterMs: 1500},
		},
		"an invalid request": {
			NewStopError(ReasonInvalidRequest),
			&v1.ProviderFailure{Kind: v1.FailureKind_FAILURE_KIND_STOP, Reason: v1.FailureReason_FAILURE_REASON_INVALID_REQUEST},
		},
		"an error of no kind": {
			errPlain,
			&v1.ProviderFailure{Kind: v1.FailureKind_FAILURE_KIND_STOP, Reason: v1.FailureReason_FAILURE_REASON_UNAVAILABLE},
		},
	} {
		t.Run(name, func(t *testing.T) {
			client := newProviderClient(t, Provider{"llm.complete": func(context.Context, Call, []byte, func([]byte) error) ([]byte, error) {
				return nil, c.err
			}})

			got, err := readCallResponse(t, client, newCallRequest("llm.complete"))

			if err != nil || got.isDone || got.failure.GetKind() != c.want.GetKind() || got.failure.GetReason() != c.want.GetReason() ||
				got.failure.GetRetryAfterMs() != c.want.GetRetryAfterMs() {
				t.Errorf("Call = %+v, %v, want %v", got, err, c.want)
			}
		})
	}
}

func TestTheTextOfAnErrorOfNoKindNeverLeavesTheProvider(t *testing.T) {
	client := newProviderClient(t, Provider{"llm.complete": func(context.Context, Call, []byte, func([]byte) error) ([]byte, error) {
		return nil, errPlain
	}})

	got, err := readCallResponse(t, client, newCallRequest("llm.complete"))

	if err != nil || strings.Contains(got.failure.String(), "sk-live") || got.failure.GetMessage() != "" {
		t.Errorf("failure = %v, %v", got.failure, err)
	}
}

func TestAnOperationTheProviderDoesNotServeIsUnimplemented(t *testing.T) {
	client := newProviderClient(t, Provider{})

	_, err := readCallResponse(t, client, newCallRequest("llm.embed"))

	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("code = %v, want unimplemented", connect.CodeOf(err))
	}
}

func TestACanceledOperationAnswersCanceled(t *testing.T) {
	client := newProviderClient(t, Provider{"llm.complete": func(context.Context, Call, []byte, func([]byte) error) ([]byte, error) {
		return nil, context.Canceled
	}})

	_, err := readCallResponse(t, client, newCallRequest("llm.complete"))

	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("code = %v, want canceled", connect.CodeOf(err))
	}
}

func TestEveryReasonOfTheSchemaIsWrittenFromOne(t *testing.T) {
	schematest.CheckProduced(t, map[v1.FailureReason]Reason{
		v1.FailureReason_FAILURE_REASON_REJECTED_KEY:     ReasonRejectedKey,
		v1.FailureReason_FAILURE_REASON_PAYMENT_REQUIRED: ReasonPaymentRequired,
		v1.FailureReason_FAILURE_REASON_RATE_LIMITED:     ReasonRateLimited,
		v1.FailureReason_FAILURE_REASON_UNAVAILABLE:      ReasonUnavailable,
		v1.FailureReason_FAILURE_REASON_INVALID_REQUEST:  ReasonInvalidRequest,
	}, writeReason)
}
