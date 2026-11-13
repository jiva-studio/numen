package connection

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/process"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// Call is the connection one call of a provider is made for. The provider keeps it, and its secrets, no
// longer than the work the call starts, such as a refresh of a list in the background, and stores no key.
//
//numen:api
type Call struct {
	// ConnectionID is the id of the connection, for the log of the provider.
	ConnectionID string
	// Type is the id of the connection type in the manifest of the plugin, without the plugin id.
	Type string
	// Fields are the fields that are not secret, with the fixed values of the type.
	Fields map[string]string
	// Secrets are the secret fields by key. A Secret prints without its value.
	Secrets map[string]Secret
	// Options are the values of the row fields of the type that the row of the call holds, such as the
	// model the person chose. The provider does not check them against its lists.
	Options map[string]string
}

// Operation carries out one operation for the connection of call. It sends its data frames in order with
// send and returns the done payload, which may be empty. It returns NewTryNextError or NewStopError for a
// call that failed, ctx.Err() for a call that was canceled, and the error of the first send that fails.
// Any other error is a stop for a reason of unavailable.
type Operation func(ctx context.Context, call Call, payload []byte, send func(data []byte) error) ([]byte, error)

// Provider holds the operations one plugin serves, by operation name, such as "llm.chat". An operation the
// map does not hold answers as not implemented.
type Provider map[string]Operation

// NewHandler creates the Connect-RPC handler for the operations of a Provider. After it answers
// connections.detected or an option list while no other call of the provider is open, it releases the
// process through releaser; a nil releaser releases nothing.
func NewHandler(provider Provider, releaser Releaser, opts ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewProviderServiceHandler(&adapter{provider: provider, calls: &openCalls{releaser: releaser}}, opts...)
}

// Mount registers the operations of a Provider on the plugin Server. It releases the process through
// the host the environment of the process names, and releases nothing where it names none.
func Mount(s *server.Server, provider Provider) {
	var releaser Releaser
	if client, err := process.NewClientFromEnv(); err == nil {
		releaser = client
	}
	s.Register(NewHandler(provider, releaser, s.HandlerOptions()...))
}

// adapter serves the operations of a Provider. A canceled call answers with the status code Canceled, and
// any other error with a failure frame; the text of an error that is no failure kind never leaves the
// provider.
type adapter struct {
	numenv1connect.UnimplementedProviderServiceHandler
	provider Provider
	calls    *openCalls
}

func (a *adapter) Call(
	ctx context.Context, request *connect.Request[v1.ProviderServiceCallRequest], stream *connect.ServerStream[v1.ProviderServiceCallResponse],
) error {
	operation, isServed := a.provider[request.Msg.GetOperation()]
	if !isServed {
		return connect.NewError(connect.CodeUnimplemented, fmt.Errorf("the operation %q is not served", request.Msg.GetOperation()))
	}
	ctx, answered := withLookup(ctx)
	a.calls.begin()
	defer a.calls.end(ctx, answered)
	var sendErr error
	send := func(data []byte) error {
		sendErr = stream.Send(&v1.ProviderServiceCallResponse{Frame: &v1.ProviderServiceCallResponse_Data{Data: data}})
		return sendErr
	}
	done, err := operation(ctx, readCall(request.Msg.GetCall()), request.Msg.GetPayload(), send)
	switch {
	case err == nil:
		return stream.Send(&v1.ProviderServiceCallResponse{Frame: &v1.ProviderServiceCallResponse_Done{Done: done}})
	case sendErr != nil:
		return sendErr
	case ctx.Err() != nil || errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, err)
	}
	return stream.Send(&v1.ProviderServiceCallResponse{Frame: &v1.ProviderServiceCallResponse_Failure{Failure: writeFailure(err)}})
}

func readCall(wire *v1.ProviderCall) Call {
	call := Call{
		ConnectionID: wire.GetConnectionId(),
		Type:         wire.GetType(),
		Fields:       wire.GetFields(),
		Secrets:      make(map[string]Secret, len(wire.GetSecrets())),
		Options:      wire.GetOptions(),
	}
	for _, one := range wire.GetSecrets() {
		call.Secrets[one.GetKey()] = Secret{Value: one.GetValue()}
	}
	return call
}

func writeFailure(err error) *v1.ProviderFailure {
	reason, _ := getReason(err)
	retryAfter, isTryNext := IsTryNext(err)
	if !isTryNext {
		return &v1.ProviderFailure{Kind: v1.FailureKind_FAILURE_KIND_STOP, Reason: writeReason(reason)}
	}
	return &v1.ProviderFailure{
		Kind:         v1.FailureKind_FAILURE_KIND_TRY_NEXT,
		Reason:       writeReason(reason),
		RetryAfterMs: retryAfter.Round(time.Millisecond).Milliseconds(),
	}
}

func writeReason(reason Reason) v1.FailureReason {
	switch reason {
	case ReasonRejectedKey:
		return v1.FailureReason_FAILURE_REASON_REJECTED_KEY
	case ReasonPaymentRequired:
		return v1.FailureReason_FAILURE_REASON_PAYMENT_REQUIRED
	case ReasonRateLimited:
		return v1.FailureReason_FAILURE_REASON_RATE_LIMITED
	case ReasonInvalidRequest:
		return v1.FailureReason_FAILURE_REASON_INVALID_REQUEST
	case ReasonUnavailable:
		return v1.FailureReason_FAILURE_REASON_UNAVAILABLE
	}
	return v1.FailureReason_FAILURE_REASON_UNAVAILABLE
}

func readReason(reason v1.FailureReason) Reason {
	switch reason {
	case v1.FailureReason_FAILURE_REASON_REJECTED_KEY:
		return ReasonRejectedKey
	case v1.FailureReason_FAILURE_REASON_PAYMENT_REQUIRED:
		return ReasonPaymentRequired
	case v1.FailureReason_FAILURE_REASON_RATE_LIMITED:
		return ReasonRateLimited
	case v1.FailureReason_FAILURE_REASON_INVALID_REQUEST:
		return ReasonInvalidRequest
	case v1.FailureReason_FAILURE_REASON_UNAVAILABLE, v1.FailureReason_FAILURE_REASON_UNSPECIFIED:
	}
	return ReasonUnavailable
}
