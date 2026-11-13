package connection

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/core/protocol/schematest"
)

// fakeHost answers a call with the frames the test sets and records the request and the token.
type fakeHost struct {
	numenv1connect.UnimplementedHostConnectionServiceHandler
	token    string
	received *v1.HostConnectionServiceCallRequest
	frames   []*v1.HostConnectionServiceCallResponse
	err      error
}

func (h *fakeHost) Call(
	_ context.Context, request *connect.Request[v1.HostConnectionServiceCallRequest],
	stream *connect.ServerStream[v1.HostConnectionServiceCallResponse],
) error {
	h.token, h.received = request.Header().Get("Authorization"), request.Msg
	for _, one := range h.frames {
		if err := stream.Send(one); err != nil {
			return err
		}
	}
	return h.err
}

func hostData(text string) *v1.HostConnectionServiceCallResponse {
	return &v1.HostConnectionServiceCallResponse{Frame: &v1.HostConnectionServiceCallResponse_Data{Data: []byte(text)}}
}

func hostDone(text string) *v1.HostConnectionServiceCallResponse {
	return &v1.HostConnectionServiceCallResponse{Frame: &v1.HostConnectionServiceCallResponse_Done{Done: []byte(text)}}
}

func hostFailure(failure *v1.CallFailure) *v1.HostConnectionServiceCallResponse {
	return &v1.HostConnectionServiceCallResponse{Frame: &v1.HostConnectionServiceCallResponse_Failure{Failure: failure}}
}

func newTestClient(t *testing.T, host *fakeHost) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostConnectionServiceHandler(host))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewClient(server.Client(), server.URL, "token-1")
}

func collectCall(t *testing.T, client *Client, priority Priority) ([]string, string, error) {
	t.Helper()
	frames, err := client.Call(t.Context(), "llm.chat", []byte("list the notes"), priority)
	if err != nil {
		return nil, "", err
	}
	data, done, err := Collect(frames)
	if err != nil {
		return nil, "", err
	}
	texts := make([]string, len(data))
	for i, one := range data {
		texts[i] = string(one)
	}
	return texts, string(done), err
}

func TestACallSendsTheOperationThePayloadAndTheTokenAndReadsTheFramesInOrder(t *testing.T) {
	host := &fakeHost{frames: []*v1.HostConnectionServiceCallResponse{hostData("Let me "), hostData("look."), hostDone("finish")}}

	frames, done, err := collectCall(t, newTestClient(t, host), PriorityInteractive)

	if err != nil || !slices.Equal(frames, []string{"Let me ", "look."}) || done != "finish" {
		t.Fatalf("Call = %v, %q, %v", frames, done, err)
	}
	if host.token != "Bearer token-1" || host.received.GetOperation() != "llm.chat" || string(host.received.GetPayload()) != "list the notes" ||
		host.received.GetPriority() != v1.Priority_PRIORITY_INTERACTIVE {
		t.Errorf("token = %q, request = %v", host.token, host.received)
	}
}

func TestAFailureFrameIsReadAsTheErrorOfTheCall(t *testing.T) {
	for name, c := range map[string]struct {
		failure *v1.CallFailure
		check   func(error) bool
	}{
		"no connection": {
			&v1.CallFailure{Error: v1.CallErrorCode_CALL_ERROR_CODE_NO_CONNECTION},
			func(err error) bool { return errors.Is(err, ErrNoConnection) },
		},
		"every row failed": {
			&v1.CallFailure{Error: v1.CallErrorCode_CALL_ERROR_CODE_ALL_FAILED, Failures: []*v1.CallRowFailure{
				{ConnectionName: "OpenRouter", Reason: "payment required", Code: v1.FailureReason_FAILURE_REASON_PAYMENT_REQUIRED},
				{ConnectionName: "Ollama", Reason: "unavailable", Code: v1.FailureReason_FAILURE_REASON_UNAVAILABLE},
			}},
			func(err error) bool {
				var failed *AllFailedError
				return errors.As(err, &failed) && slices.Equal(failed.Failures, []Failure{
					{ConnectionName: "OpenRouter", Reason: "payment required", Code: ReasonPaymentRequired},
					{ConnectionName: "Ollama", Reason: "unavailable", Code: ReasonUnavailable},
				})
			},
		},
		"a stop": {
			&v1.CallFailure{Error: v1.CallErrorCode_CALL_ERROR_CODE_STOPPED, Failures: []*v1.CallRowFailure{
				{ConnectionName: "OpenRouter", Reason: "invalid request", Code: v1.FailureReason_FAILURE_REASON_INVALID_REQUEST},
			}},
			func(err error) bool {
				var stopped *StoppedError
				return errors.As(err, &stopped) && stopped.ConnectionName == "OpenRouter" && stopped.Reason == "invalid request" &&
					stopped.Code == ReasonInvalidRequest
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			host := &fakeHost{frames: []*v1.HostConnectionServiceCallResponse{hostData("partial"), hostFailure(c.failure)}}

			frames, _, err := collectCall(t, newTestClient(t, host), PriorityBackground)

			if !c.check(err) || frames != nil {
				t.Errorf("err = %v, frames = %v", err, frames)
			}
		})
	}
}

func TestAStreamThatEndsWithoutAFinalFrameIsAnError(t *testing.T) {
	host := &fakeHost{frames: []*v1.HostConnectionServiceCallResponse{hostData("partial")}}

	_, _, err := collectCall(t, newTestClient(t, host), PriorityBackground)

	if !errors.Is(err, ErrStreamEnded) {
		t.Errorf("err = %v, want ErrStreamEnded", err)
	}
}

func TestAStatusOfTheHostIsAnErrorOfNoFailureKind(t *testing.T) {
	host := &fakeHost{err: connect.NewError(connect.CodeInternal, errors.New("the settings are unreadable"))}

	_, _, err := collectCall(t, newTestClient(t, host), PriorityBackground)

	var failed *AllFailedError
	if err == nil || errors.Is(err, ErrNoConnection) || errors.As(err, &failed) {
		t.Errorf("err = %v, want an error of no failure kind", err)
	}
}

func TestACanceledCallReturnsTheErrorOfTheContext(t *testing.T) {
	client := newTestClient(t, &fakeHost{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	frames, err := client.Call(ctx, "llm.complete", nil, PriorityBackground)
	if err == nil {
		_, _, err = Collect(frames)
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestEveryErrorCodeOfTheSchemaIsReadAsAnError(t *testing.T) {
	schematest.CheckHandled(t, func(code v1.CallErrorCode) bool {
		err := readFailure(&v1.CallFailure{Error: code})
		var failed *AllFailedError
		var stopped *StoppedError
		return errors.Is(err, ErrNoConnection) || errors.As(err, &failed) || errors.As(err, &stopped)
	})
}

func TestEveryPriorityOfTheSchemaIsNumberedFromOne(t *testing.T) {
	schematest.CheckProduced(t, map[v1.Priority]Priority{
		v1.Priority_PRIORITY_INTERACTIVE: PriorityInteractive,
		v1.Priority_PRIORITY_BACKGROUND:  PriorityBackground,
	}, writePriority)
}

func TestEveryFailureReasonOfTheSchemaIsReadAsTheCodeOfAFailure(t *testing.T) {
	want := map[v1.FailureReason]Reason{
		v1.FailureReason_FAILURE_REASON_UNAVAILABLE:      ReasonUnavailable,
		v1.FailureReason_FAILURE_REASON_REJECTED_KEY:     ReasonRejectedKey,
		v1.FailureReason_FAILURE_REASON_PAYMENT_REQUIRED: ReasonPaymentRequired,
		v1.FailureReason_FAILURE_REASON_RATE_LIMITED:     ReasonRateLimited,
		v1.FailureReason_FAILURE_REASON_INVALID_REQUEST:  ReasonInvalidRequest,
	}
	schematest.CheckHandled(t, func(code v1.FailureReason) bool {
		err := readFailure(&v1.CallFailure{
			Error: v1.CallErrorCode_CALL_ERROR_CODE_ALL_FAILED, Failures: []*v1.CallRowFailure{{ConnectionName: "Harbor", Reason: "words", Code: code}},
		})
		var failed *AllFailedError
		return errors.As(err, &failed) && failed.Failures[0].Code == want[code]
	})
}
