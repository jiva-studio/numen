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
			Preferred: wire.ConvertPreferredRowFromWire(request.Msg.GetPreferred()),
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

// ListRows lists the rows of the calling plugin for the operation. A host with no calls lists none.
func (e *connectionEndpoint) ListRows(
	ctx context.Context, request *connect.Request[v1.HostConnectionServiceListRowsRequest],
) (*connect.Response[v1.HostConnectionServiceListRowsResponse], error) {
	pluginID, err := e.endpoint.authenticate(request.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	calls := e.endpoint.callers.Lent.GetModelCalls()
	if calls == nil {
		return connect.NewResponse(&v1.HostConnectionServiceListRowsResponse{}), nil
	}
	rows, err := calls.ListRows(pluginID, request.Msg.GetOperation())
	if err != nil {
		return nil, wire.NewRowListError(ctx, err)
	}
	return connect.NewResponse(&v1.HostConnectionServiceListRowsResponse{Rows: wire.ConvertCallRowsToWire(rows)}), nil
}

// ListRowOptions lists the values a row field of a connection among the rows of the calling plugin can take.
// A host with no calls finds no connection.
func (e *connectionEndpoint) ListRowOptions(
	ctx context.Context, request *connect.Request[v1.HostConnectionServiceListRowOptionsRequest],
) (*connect.Response[v1.HostConnectionServiceListRowOptionsResponse], error) {
	pluginID, err := e.endpoint.authenticate(request.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	calls := e.endpoint.callers.Lent.GetModelCalls()
	if calls == nil {
		return nil, wire.NewRowListError(ctx, connection.ErrNotFound)
	}
	message := request.Msg
	options, err := calls.ListRowOptions(
		ctx, pluginID, message.GetOperation(), message.GetConnectionId(), message.GetField(), message.GetOptions(),
	)
	if err != nil {
		return nil, wire.NewRowListError(ctx, err)
	}
	return connect.NewResponse(&v1.HostConnectionServiceListRowOptionsResponse{Options: wire.ConvertOptionsToWire(options)}), nil
}
