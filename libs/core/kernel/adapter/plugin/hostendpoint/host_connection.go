package hostendpoint

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// connectionEndpoint answers HostConnectionService for the plugins that call the host. The connections are
// the rows of the calling plugin for the operation, so a plugin reaches only the operations its settings of
// rows name, and a call names no connection.
type connectionEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostConnectionServiceHandler = (*connectionEndpoint)(nil)

// Call streams the frames of one call. A call the core could not serve ends the stream with a failure
// frame that holds the error code; any other failure is a status code. A host with no calls answers every
// call with connection.ErrNoConnection.
func (e *connectionEndpoint) Call(
	ctx context.Context, request *connect.Request[v1.HostConnectionServiceCallRequest],
	stream *connect.ServerStream[v1.HostConnectionServiceCallResponse],
) error {
	pluginID, err := e.endpoint.authenticate(request.Header().Get("Authorization"))
	if err != nil {
		return err
	}
	calls := e.endpoint.callers.Lent.GetModelCalls()
	var done []byte
	err = connection.ErrNoConnection
	if calls != nil {
		done, err = calls.Call(ctx, pluginID, modelcall.Request{
			Operation: request.Msg.GetOperation(), Payload: request.Msg.GetPayload(), Priority: readPriority(request.Msg.GetPriority()),
		}, func(data []byte) error {
			return stream.Send(&v1.HostConnectionServiceCallResponse{Frame: &v1.HostConnectionServiceCallResponse_Data{Data: data}})
		})
	}
	if err == nil {
		return stream.Send(&v1.HostConnectionServiceCallResponse{Frame: &v1.HostConnectionServiceCallResponse_Done{Done: done}})
	}
	failure, isServed := writeCallFailure(ctx, err)
	if !isServed {
		return readCallStatus(ctx, err)
	}
	return stream.Send(&v1.HostConnectionServiceCallResponse{Frame: &v1.HostConnectionServiceCallResponse_Failure{Failure: failure}})
}

func writeCallFailure(ctx context.Context, err error) (*v1.CallFailure, bool) {
	if ctx.Err() != nil {
		return nil, false
	}
	var allFailed *connection.AllFailedError
	var stopped *connection.StoppedError
	switch {
	case errors.Is(err, connection.ErrNoConnection):
		return &v1.CallFailure{Error: v1.CallErrorCode_CALL_ERROR_CODE_NO_CONNECTION}, true
	case errors.As(err, &allFailed):
		failures := make([]*v1.CallRowFailure, len(allFailed.Failures))
		for i, one := range allFailed.Failures {
			failures[i] = &v1.CallRowFailure{ConnectionName: one.ConnectionName, Reason: one.Reason}
		}
		return &v1.CallFailure{Error: v1.CallErrorCode_CALL_ERROR_CODE_ALL_FAILED, Detail: allFailed.Error(), Failures: failures}, true
	case errors.As(err, &stopped):
		return &v1.CallFailure{
			Error: v1.CallErrorCode_CALL_ERROR_CODE_STOPPED, Detail: stopped.Error(),
			Failures: []*v1.CallRowFailure{{ConnectionName: stopped.ConnectionName, Reason: stopped.Reason}},
		}, true
	}
	return nil, false
}

func readCallStatus(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return connect.NewError(connect.CodeCanceled, ctxErr)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func readPriority(priority v1.Priority) modelcall.Priority {
	if priority == v1.Priority_PRIORITY_INTERACTIVE {
		return modelcall.PriorityInteractive
	}
	return modelcall.PriorityBackground
}
