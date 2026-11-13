package llmtest

import (
	"context"
	"net/http"
	"net/http/httptest"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

// NewCaller returns a Caller that carries out each operation with the operation of provider, for call, in
// the process of the test, as the core carries it out for one row: through the handler of
// connection.NewHandler and its failure frames. It runs an operation to its end before it returns the
// frames. The frames end with the errors of connection.Frames.Err, which name the connection by
// call.ConnectionID: a stop, or an error that is no failure kind, is a *StoppedError; a try-next, or a
// failure after the first data frame, is an *AllFailedError with one failure; an operation the provider
// does not serve is ErrNoConnection; a canceled call is the error of the context. Unlike the core, the
// caller tries a row that is unavailable only once.
func NewCaller(provider connection.Provider, call connection.Call) connection.Caller {
	return NewReleasingCaller(provider, call, nil)
}

// NewReleasingCaller returns the Caller of NewCaller over a handler that releases the process through
// releaser, as connection.Mount does through the host.
func NewReleasingCaller(provider connection.Provider, call connection.Call, releaser connection.Releaser) connection.Caller {
	_, handler := connection.NewHandler(provider, releaser)
	client := numenv1connect.NewProviderServiceClient(&http.Client{Transport: handlerTransport{handler: handler}}, "http://llmtest")
	return &caller{client: client, call: call}
}

type caller struct {
	client numenv1connect.ProviderServiceClient
	call   connection.Call
}

func (c *caller) Call(ctx context.Context, operation string, payload []byte, _ connection.Priority) (connection.Frames, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	stream, err := c.client.Call(ctx, connect.NewRequest(&v1.ProviderServiceCallRequest{
		Operation: operation, Call: writeCall(c.call), Payload: payload,
	}))
	if err != nil {
		return nil, err
	}
	frames := &frames{next: -1}
	for stream.Receive() {
		switch frame := stream.Msg().GetFrame().(type) {
		case *v1.ProviderServiceCallResponse_Data:
			frames.data = append(frames.data, frame.Data)
		case *v1.ProviderServiceCallResponse_Done:
			frames.done = frame.Done
		case *v1.ProviderServiceCallResponse_Failure:
			frames.err = c.readFailure(frame.Failure, len(frames.data) > 0)
		}
	}
	if err := stream.Err(); err != nil {
		frames.err = readStreamError(ctx, err)
	}
	if err := stream.Close(); err != nil && frames.err == nil {
		frames.err = err
	}
	return frames, nil
}

func (c *caller) readFailure(failure *v1.ProviderFailure, hasData bool) error {
	code := readReason(failure.GetReason())
	if failure.GetKind() == v1.FailureKind_FAILURE_KIND_TRY_NEXT || hasData {
		return &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: c.call.ConnectionID, Reason: code.String(), Code: code}}}
	}
	return &connection.StoppedError{ConnectionName: c.call.ConnectionID, Reason: code.String(), Code: code}
}

func readStreamError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	switch connect.CodeOf(err) {
	case connect.CodeUnimplemented:
		return connection.ErrNoConnection
	case connect.CodeCanceled:
		return context.Canceled
	default:
		return err
	}
}

func readReason(reason v1.FailureReason) connection.Reason {
	switch reason {
	case v1.FailureReason_FAILURE_REASON_REJECTED_KEY:
		return connection.ReasonRejectedKey
	case v1.FailureReason_FAILURE_REASON_PAYMENT_REQUIRED:
		return connection.ReasonPaymentRequired
	case v1.FailureReason_FAILURE_REASON_RATE_LIMITED:
		return connection.ReasonRateLimited
	case v1.FailureReason_FAILURE_REASON_INVALID_REQUEST:
		return connection.ReasonInvalidRequest
	case v1.FailureReason_FAILURE_REASON_UNAVAILABLE, v1.FailureReason_FAILURE_REASON_UNSPECIFIED:
		return connection.ReasonUnavailable
	}
	return connection.ReasonUnavailable
}

func writeCall(call connection.Call) *v1.ProviderCall {
	secrets := make([]*v1.ProviderSecret, 0, len(call.Secrets))
	for key, secret := range call.Secrets {
		secrets = append(secrets, &v1.ProviderSecret{Key: key, Value: secret.Value})
	}
	return &v1.ProviderCall{
		ConnectionId: call.ConnectionID, Type: call.Type, Fields: call.Fields, Secrets: secrets, Options: call.Options,
	}
}

// handlerTransport serves each request with handler in the process of the test, without a network.
type handlerTransport struct {
	handler http.Handler
}

func (t handlerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	t.handler.ServeHTTP(recorder, request)
	return recorder.Result(), nil
}

// frames are the frames an operation sent, and its done payload or its error.
type frames struct {
	data [][]byte
	next int
	done []byte
	err  error
}

func (f *frames) Next() bool {
	if f.next+1 >= len(f.data) {
		f.next = len(f.data)
		return false
	}
	f.next++
	return true
}

func (f *frames) Data() []byte {
	if f.next < 0 || f.next >= len(f.data) {
		return nil
	}
	return f.data[f.next]
}

func (f *frames) Done() []byte { return f.done }

func (f *frames) Err() error { return f.err }

func (f *frames) Close() error { return nil }
