package connection

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/shared/protocol/schematest"
)

// fakeHost answers a call with the frames the test sets and records the request and the token.
type fakeHost struct {
	numenv1connect.UnimplementedConnectionCallServiceHandler
	token    string
	received *v1.ConnectionCallServiceCallRequest
	frames   []*v1.ConnectionCallServiceCallResponse
	onCall   func()
	err      error
}

func (h *fakeHost) Call(
	_ context.Context, request *connect.Request[v1.ConnectionCallServiceCallRequest],
	stream *connect.ServerStream[v1.ConnectionCallServiceCallResponse],
) error {
	h.token, h.received = request.Header().Get("Authorization"), request.Msg
	if h.onCall != nil {
		h.onCall()
	}
	for _, one := range h.frames {
		if err := stream.Send(one); err != nil {
			return err
		}
	}
	return h.err
}

func hostData(text string) *v1.ConnectionCallServiceCallResponse {
	return &v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Data{Data: []byte(text)}}
}

func hostDone(text string) *v1.ConnectionCallServiceCallResponse {
	return &v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Done{Done: []byte(text)}}
}

func hostFailure(failure *v1.CallFailure) *v1.ConnectionCallServiceCallResponse {
	return &v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Failure{Failure: failure}}
}

func newTestClient(t *testing.T, host *fakeHost) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewConnectionCallServiceHandler(host))
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
	host := &fakeHost{frames: []*v1.ConnectionCallServiceCallResponse{hostData("Let me "), hostData("look."), hostDone("finish")}}

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
			&v1.CallFailure{Error: v1.ErrorCode_ERROR_CODE_NO_CONNECTION},
			func(err error) bool { return errors.Is(err, ErrNoConnection) },
		},
		"every row failed": {
			&v1.CallFailure{Error: v1.ErrorCode_ERROR_CODE_ALL_FAILED, Failures: []*v1.CallRowFailure{
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
			&v1.CallFailure{Error: v1.ErrorCode_ERROR_CODE_STOPPED, Failures: []*v1.CallRowFailure{
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
			host := &fakeHost{frames: []*v1.ConnectionCallServiceCallResponse{hostData("partial"), hostFailure(c.failure)}}

			frames, _, err := collectCall(t, newTestClient(t, host), PriorityBackground)

			if !c.check(err) || frames != nil {
				t.Errorf("err = %v, frames = %v", err, frames)
			}
		})
	}
}

func TestAStreamThatEndsWithoutAFinalFrameIsAnError(t *testing.T) {
	host := &fakeHost{frames: []*v1.ConnectionCallServiceCallResponse{hostData("partial")}}

	_, _, err := collectCall(t, newTestClient(t, host), PriorityBackground)

	if !errors.Is(err, ErrStreamEnded) {
		t.Errorf("err = %v, want ErrStreamEnded", err)
	}
}

func TestAStatusOfTheHostIsAnErrorOfNoFailureKind(t *testing.T) {
	host := &fakeHost{err: connect.NewError(connect.CodeInternal, errors.New("the settings are unreadable"))}

	_, _, err := collectCall(t, newTestClient(t, host), PriorityBackground)

	var failed *AllFailedError
	if connect.CodeOf(err) != connect.CodeInternal || errors.Is(err, ErrNoConnection) || errors.As(err, &failed) {
		t.Errorf("err = %v, want CodeInternal of no failure kind", err)
	}
}

type failingCloseBody struct {
	io.ReadCloser
	closeErr error
}

func (b *failingCloseBody) Close() error {
	return errors.Join(b.ReadCloser.Close(), b.closeErr)
}

type failingCloseTransport struct {
	base     http.RoundTripper
	closeErr error
}

func (c *failingCloseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Body = &failingCloseBody{ReadCloser: resp.Body, closeErr: c.closeErr}
	return resp, nil
}

func TestACanceledCallReturnsTheErrorOfTheContext(t *testing.T) {
	client := newTestClient(t, &fakeHost{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := client.Call(ctx, "llm.complete", nil, PriorityBackground); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}

	streamCtx, cancelStream := context.WithCancel(t.Context())
	streamHost := &fakeHost{
		frames: []*v1.ConnectionCallServiceCallResponse{hostData("first")},
		err:    connect.NewError(connect.CodeInternal, errors.New("aborted")),
	}
	frames, err := newTestClient(t, streamHost).Call(streamCtx, "llm.complete", nil, PriorityBackground)
	if err != nil {
		t.Fatal(err)
	}
	if !frames.Next() {
		t.Fatal("expected first frame")
	}
	cancelStream()
	if frames.Next() || !errors.Is(frames.Err(), context.Canceled) {
		t.Errorf("stream err = %v, want context.Canceled", frames.Err())
	}
	if err := frames.Close(); err != nil {
		t.Fatal(err)
	}

	doneHost := &fakeHost{frames: []*v1.ConnectionCallServiceCallResponse{hostDone("finish"), hostData("trailing")}}
	doneFrames, err := newTestClient(t, doneHost).Call(t.Context(), "llm.complete", nil, PriorityBackground)
	if err != nil {
		t.Fatal(err)
	}
	firstDone := doneFrames.Next()
	secondDone := doneFrames.Next()
	if firstDone || secondDone || string(doneFrames.Done()) != "finish" {
		t.Errorf("doneFrames = %q, want finish and no more frames", doneFrames.Done())
	}
	if err := doneFrames.Close(); err != nil {
		t.Fatal(err)
	}

	failHost := &fakeHost{frames: []*v1.ConnectionCallServiceCallResponse{
		hostFailure(&v1.CallFailure{Error: v1.ErrorCode_ERROR_CODE_NO_CONNECTION}),
		hostData("trailing"),
	}}
	failFrames, err := newTestClient(t, failHost).Call(t.Context(), "llm.complete", nil, PriorityBackground)
	if err != nil {
		t.Fatal(err)
	}
	firstFail := failFrames.Next()
	secondFail := failFrames.Next()
	if firstFail || secondFail || !errors.Is(failFrames.Err(), ErrNoConnection) {
		t.Errorf("failFrames err = %v, want ErrNoConnection and no more frames", failFrames.Err())
	}
	if err := failFrames.Close(); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewConnectionCallServiceHandler(&fakeHost{
		frames: []*v1.ConnectionCallServiceCallResponse{hostData("chunk"), hostDone("ok")},
	}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	closeErr := errors.New("close failed")
	closingClient := NewClient(&http.Client{Transport: &failingCloseTransport{base: server.Client().Transport, closeErr: closeErr}}, server.URL, "token-1")
	closingFrames, err := closingClient.Call(t.Context(), "llm.chat", nil, PriorityBackground)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Collect(closingFrames); !errors.Is(err, closeErr) {
		t.Errorf("Collect close err = %v, want %v", err, closeErr)
	}
}

func TestEveryCallErrorCodeIsReadAsAnError(t *testing.T) {
	for _, code := range []v1.ErrorCode{
		v1.ErrorCode_ERROR_CODE_NO_CONNECTION,
		v1.ErrorCode_ERROR_CODE_ALL_FAILED,
		v1.ErrorCode_ERROR_CODE_STOPPED,
	} {
		err := readFailure(&v1.CallFailure{Error: code})
		var failed *AllFailedError
		var stopped *StoppedError
		if !errors.Is(err, ErrNoConnection) && !errors.As(err, &failed) && !errors.As(err, &stopped) {
			t.Errorf("%v reaches nothing on the way in", code)
		}
	}
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
			Error: v1.ErrorCode_ERROR_CODE_ALL_FAILED, Failures: []*v1.CallRowFailure{{ConnectionName: "Harbor", Reason: "words", Code: code}},
		})
		var failed *AllFailedError
		return errors.As(err, &failed) && failed.Failures[0].Code == want[code]
	})
}
