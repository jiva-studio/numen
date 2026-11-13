package hostendpoint

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
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
			Operation: request.Msg.GetOperation(), Payload: request.Msg.GetPayload(), Priority: wire.ConvertPriorityFromWire(request.Msg.GetPriority()),
		}, func(data []byte) error {
			return stream.Send(&v1.HostConnectionServiceCallResponse{Frame: &v1.HostConnectionServiceCallResponse_Data{Data: data}})
		})
	}
	if err == nil {
		return stream.Send(&v1.HostConnectionServiceCallResponse{Frame: &v1.HostConnectionServiceCallResponse_Done{Done: done}})
	}
	failure, isServed := wire.ConvertCallFailureToWire(ctx, err)
	if !isServed {
		return wire.NewCallStatusError(ctx, err)
	}
	return stream.Send(&v1.HostConnectionServiceCallResponse{Frame: &v1.HostConnectionServiceCallResponse_Failure{Failure: failure}})
}
