// Package connectioncall serves a window's ConnectionCallService: the operations a plugin view asks for,
// through the rows of the plugin the request names.
package connectioncall

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

var errPluginToken = errors.New("ConnectionCallService is not served to a plugin")

// Calls carries out the operations plugins ask for.
type Calls interface {
	Call(ctx context.Context, callerPluginID string, request modelcall.Request, send modelcall.Send) ([]byte, error)
	ListRows(callerPluginID, operation string) ([]modelcall.Row, error)
	ListRowOptions(ctx context.Context, callerPluginID, operation, connectionID, field string, options map[string]string) ([]modelcall.Option, error)
}

// Handler serves ConnectionCallService over the calls.
type Handler struct {
	calls Calls
}

// NewHandler is the handler over the calls of this installation.
func NewHandler(calls Calls) *Handler {
	return &Handler{calls: calls}
}

func refuseToken(header http.Header) error {
	// A window sends no token; a plugin process sends the one the host issued it, and these calls name the
	// plugin they act for in the request.
	if header.Get("Authorization") != "" {
		return connect.NewError(connect.CodePermissionDenied, errPluginToken)
	}
	return nil
}

// Call streams the frames of one call through the rows of the plugin the request names. A call the core
// could not serve ends the stream with a failure frame that holds the error code.
func (h *Handler) Call(
	ctx context.Context, request *connect.Request[v1.ConnectionCallServiceCallRequest],
	stream *connect.ServerStream[v1.ConnectionCallServiceCallResponse],
) error {
	if err := refuseToken(request.Header()); err != nil {
		return err
	}
	done, err := h.calls.Call(ctx, request.Msg.GetPluginId(), modelcall.Request{
		Operation: request.Msg.GetOperation(), Payload: request.Msg.GetPayload(), Priority: wire.ConvertPriorityFromWire(request.Msg.GetPriority()),
		Preferred: wire.ConvertPreferredRowFromWire(request.Msg.GetPreferred()),
	}, func(data []byte) error {
		return stream.Send(&v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Data{Data: data}})
	})
	if err == nil {
		return stream.Send(&v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Done{Done: done}})
	}
	failure, isServed := wire.ConvertCallFailureToWire(ctx, err)
	if !isServed {
		return wire.NewCallStatusError(ctx, err)
	}
	return stream.Send(&v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Failure{Failure: failure}})
}

// ListRows lists the rows of the plugin the request names for the operation.
func (h *Handler) ListRows(
	ctx context.Context, request *connect.Request[v1.ConnectionCallServiceListRowsRequest],
) (*connect.Response[v1.ConnectionCallServiceListRowsResponse], error) {
	if err := refuseToken(request.Header()); err != nil {
		return nil, err
	}
	rows, err := h.calls.ListRows(request.Msg.GetPluginId(), request.Msg.GetOperation())
	if err != nil {
		return nil, wire.NewRowListError(ctx, err)
	}
	return connect.NewResponse(&v1.ConnectionCallServiceListRowsResponse{Rows: wire.ConvertCallRowsToWire(rows)}), nil
}

// ListRowOptions lists the values a row field of a connection among the rows of the plugin can take.
func (h *Handler) ListRowOptions(
	ctx context.Context, request *connect.Request[v1.ConnectionCallServiceListRowOptionsRequest],
) (*connect.Response[v1.ConnectionCallServiceListRowOptionsResponse], error) {
	if err := refuseToken(request.Header()); err != nil {
		return nil, err
	}
	message := request.Msg
	options, err := h.calls.ListRowOptions(
		ctx, message.GetPluginId(), message.GetOperation(), message.GetConnectionId(), message.GetField(), message.GetOptions(),
	)
	if err != nil {
		return nil, wire.NewRowListError(ctx, err)
	}
	return connect.NewResponse(&v1.ConnectionCallServiceListRowOptionsResponse{Options: wire.ConvertOptionsToWire(options)}), nil
}
