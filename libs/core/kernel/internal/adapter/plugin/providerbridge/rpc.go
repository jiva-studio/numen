package providerbridge

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

var errStreamEnded = errors.New("the stream of the provider ended with neither a done nor a failure")

// Source opens the provider of a plugin over its ProviderService, with a use lease or, where isQuestion is
// set, a question lease.
type Source struct {
	host       bridge.Activator
	isQuestion bool
}

// NewSource returns the source over the plugins the host activates, which holds a plugin for a use of it.
func NewSource(host bridge.Activator) *Source { return &Source{host: host, isQuestion: false} }

// NewQuestionSource returns the source over the plugins the host activates, which holds a plugin for a
// question the core asks it for itself.
func NewQuestionSource(host bridge.Activator) *Source { return &Source{host: host, isQuestion: true} }

// OpenProvider activates the plugin, which starts it if it is not running, and returns its provider with
// the function that releases the plugin. The function may be called more than once.
func (s *Source) OpenProvider(ctx context.Context, pluginID string) (connectionport.Provider, func(), error) {
	open := bridge.Open[numenv1connect.ProviderServiceClient]
	if s.isQuestion {
		open = bridge.OpenForQuestion[numenv1connect.ProviderServiceClient]
	}
	client, lease, err := open(ctx, s.host, pluginID, numenv1connect.NewProviderServiceClient)
	if err != nil {
		return nil, nil, err
	}
	return &provider{client: client}, lease.Release, nil
}

// provider carries the calls of the core to the ProviderService of a plugin. An operation the plugin does
// not serve is a stop, and the end of the context is returned as it is.
type provider struct {
	client numenv1connect.ProviderServiceClient
}

// Call streams one call from the provider and sends each data frame as it arrives. A failure the provider
// reports ends the call as *TryNextError or *StopError, the error of send ends it as it is, and a stream
// that ends with neither a done nor a failure is an error of no failure kind.
func (p *provider) Call(
	ctx context.Context, operation string, call modelcall.ProviderCall, payload []byte, send modelcall.Send,
) (done []byte, err error) {
	stream, err := p.client.Call(ctx, connect.NewRequest(&v1.ProviderServiceCallRequest{
		Operation: operation, Call: writeCall(call), Payload: payload,
	}))
	if err != nil {
		return nil, readCallError(ctx, err)
	}
	defer func() { err = errors.Join(err, stream.Close()) }()
	for stream.Receive() {
		switch frame := stream.Msg().GetFrame().(type) {
		case *v1.ProviderServiceCallResponse_Data:
			if err := send(frame.Data); err != nil {
				return nil, err
			}
		case *v1.ProviderServiceCallResponse_Done:
			return frame.Done, nil
		case *v1.ProviderServiceCallResponse_Failure:
			return nil, readFailure(frame.Failure)
		}
	}
	if streamErr := stream.Err(); streamErr != nil {
		return nil, readCallError(ctx, streamErr)
	}
	return nil, errStreamEnded
}

// ListOptions calls the operation with a numen.v1.OptionsRequest and reads the numen.v1.Options of its done
// frame. A done frame that holds no Options is an invalid request.
func (p *provider) ListOptions(
	ctx context.Context, operation string, call modelcall.ProviderCall, operations []string,
) ([]modelcall.Option, error) {
	payload, err := proto.Marshal(&v1.OptionsRequest{Operations: operations})
	if err != nil {
		return nil, err
	}
	done, err := p.Call(ctx, operation, call, payload, func([]byte) error { return nil })
	if err != nil {
		return nil, err
	}
	var listed v1.Options
	if err := proto.Unmarshal(done, &listed); err != nil {
		return nil, &connection.StopError{Reason: connection.FailureReasonInvalidRequest}
	}
	options := make([]modelcall.Option, len(listed.GetOptions()))
	for i, one := range listed.GetOptions() {
		options[i] = modelcall.Option{Value: one.GetValue(), Title: one.GetTitle(), Group: one.GetGroup()}
	}
	return options, nil
}

// ListDetected calls connections.detected with an empty payload and reads the numen.v1.DetectedConnections of its
// done frame. A done frame that holds no DetectedConnections is an invalid request.
func (p *provider) ListDetected(ctx context.Context, call modelcall.ProviderCall) ([]modelcall.DetectedConnection, error) {
	done, err := p.Call(ctx, connection.OperationDetected, call, nil, func([]byte) error { return nil })
	if err != nil {
		return nil, err
	}
	var listed v1.DetectedConnections
	if err := proto.Unmarshal(done, &listed); err != nil {
		return nil, &connection.StopError{Reason: connection.FailureReasonInvalidRequest}
	}
	found := make([]modelcall.DetectedConnection, len(listed.GetConnections()))
	for i, one := range listed.GetConnections() {
		found[i] = modelcall.DetectedConnection{Name: one.GetName(), Fields: one.GetFields()}
	}
	return found, nil
}

func writeCall(call modelcall.ProviderCall) *v1.ProviderCall {
	secrets := make([]*v1.ProviderSecret, 0, len(call.Secrets))
	for key, value := range call.Secrets {
		secrets = append(secrets, &v1.ProviderSecret{Key: key, Value: value})
	}
	return &v1.ProviderCall{
		ConnectionId: call.ConnectionID, Type: call.Type, Fields: call.Fields, Secrets: secrets, Options: call.Options,
	}
}

func readCallError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if connect.CodeOf(err) == connect.CodeUnimplemented {
		return &connection.StopError{Reason: connection.FailureReasonInvalidRequest}
	}
	return err
}

func readFailure(failure *v1.ProviderFailure) error {
	reason := readReason(failure.GetReason())
	if failure.GetKind() == v1.FailureKind_FAILURE_KIND_TRY_NEXT {
		return &connection.TryNextError{Reason: reason, RetryAfter: time.Duration(max(failure.GetRetryAfterMs(), 0)) * time.Millisecond}
	}
	return &connection.StopError{Reason: reason}
}

func readReason(reason v1.FailureReason) connection.FailureReason {
	switch reason {
	case v1.FailureReason_FAILURE_REASON_REJECTED_KEY:
		return connection.FailureReasonRejectedKey
	case v1.FailureReason_FAILURE_REASON_PAYMENT_REQUIRED:
		return connection.FailureReasonPaymentRequired
	case v1.FailureReason_FAILURE_REASON_RATE_LIMITED:
		return connection.FailureReasonRateLimited
	case v1.FailureReason_FAILURE_REASON_INVALID_REQUEST:
		return connection.FailureReasonInvalidRequest
	case v1.FailureReason_FAILURE_REASON_UNAVAILABLE, v1.FailureReason_FAILURE_REASON_UNSPECIFIED:
		return connection.FailureReasonUnavailable
	}
	return connection.FailureReasonUnavailable
}
