// Package connectioncall answers ConnectionCallService for both endpoints that serve it: the window, whose
// request names the plugin, and the host endpoint, where the token names it.
package connectioncall

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
)

// Calls carries out the operations plugins ask for. A nil Calls answers every call with
// connection.ErrNoConnection.
type Calls interface {
	Call(ctx context.Context, callerPluginID string, request modelcall.Request, send modelcall.Send) ([]byte, error)
}

// Rows lists the rows of the plugins and the values of their row fields. A nil Rows lists no row and finds no
// connection.
type Rows interface {
	ListRows(callerPluginID, operation string) ([]modelcall.Row, error)
	ListRowOptions(ctx context.Context, callerPluginID, operation, connectionID, field string, options map[string]string) ([]modelcall.Option, error)
}

// Cancels holds the cancel of each running call under its key.
type Cancels interface {
	// Start returns the context of the call, which Cancel with the same key cancels, and the function that
	// ends the call.
	Start(ctx context.Context, key string) (context.Context, func())
	Cancel(key string)
}

// Identify tells the plugin a request is made for from its headers and the plugin_id it names, which is empty
// where the request names none, and returns the context the request is answered in.
type Identify func(ctx context.Context, header http.Header, requested string) (context.Context, string, error)

// Handler serves ConnectionCallService over the calls and the rows the sources return at each request, for
// the plugin that identify names.
type Handler struct {
	calls    func() Calls
	rows     func() Rows
	identify Identify
	cancels  Cancels
}

// NewHandler is the handler over the calls and rows its sources return, for the plugin identify names.
// cancels holds the cancels of the calls it serves.
func NewHandler(calls func() Calls, rows func() Rows, identify Identify, cancels Cancels) *Handler {
	return &Handler{calls: calls, rows: rows, identify: identify, cancels: cancels}
}

// Call streams the frames of one call through the rows of the plugin. A call the core could not serve ends
// the stream with a failure frame that holds the error code; any other failure is a status code.
func (h *Handler) Call(
	ctx context.Context, request *connect.Request[v1.ConnectionCallServiceCallRequest],
	stream *connect.ServerStream[v1.ConnectionCallServiceCallResponse],
) error {
	ctx, pluginID, err := h.identify(ctx, request.Header(), request.Msg.GetPluginId())
	if err != nil {
		return err
	}
	ctx, end := h.cancels.Start(ctx, getCancelKey(pluginID, request.Msg.GetCallId()))
	defer end()
	if err := ctx.Err(); err != nil {
		return connect.NewError(connect.CodeCanceled, err)
	}
	return wire.RunWatch(ctx, stream, wire.ResendInterval, nil, func(ctx context.Context, send func(*v1.ConnectionCallServiceCallResponse) error) error {
		var done []byte
		err := connection.ErrNoConnection
		if calls := h.calls(); calls != nil {
			done, err = calls.Call(ctx, pluginID, modelcall.Request{
				Operation: request.Msg.GetOperation(), Payload: request.Msg.GetPayload(),
				Priority:  wire.ConvertPriorityFromWire(request.Msg.GetPriority()),
				Preferred: wire.ConvertPreferredRowFromWire(request.Msg.GetPreferredRow()),
			}, func(data []byte) error {
				return send(&v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Data{Data: data}})
			})
		}
		if err == nil {
			return send(&v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Done{Done: done}})
		}
		failure, isServed := wire.ConvertCallFailureToWire(ctx, err)
		if !isServed {
			return wire.NewCallStatusError(ctx, err)
		}
		return send(&v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Failure{Failure: failure}})
	})
}

// CancelCall cancels the context of the call of the plugin that carries the call id. The caller sends it when
// it aborts the call, because an aborted request does not end the context of the call.
func (h *Handler) CancelCall(
	ctx context.Context, request *connect.Request[v1.CancelCallRequest],
) (*connect.Response[v1.CancelCallResponse], error) {
	_, pluginID, err := h.identify(ctx, request.Header(), request.Msg.GetPluginId())
	if err != nil {
		return nil, err
	}
	h.cancels.Cancel(getCancelKey(pluginID, request.Msg.GetCallId()))
	return connect.NewResponse(&v1.CancelCallResponse{}), nil
}

// ListRows lists the rows of the plugin for the operation. A host with no rows lists none.
func (h *Handler) ListRows(
	ctx context.Context, request *connect.Request[v1.ConnectionCallServiceListRowsRequest],
) (*connect.Response[v1.ConnectionCallServiceListRowsResponse], error) {
	ctx, pluginID, err := h.identify(ctx, request.Header(), request.Msg.GetPluginId())
	if err != nil {
		return nil, err
	}
	var rows []modelcall.Row
	if lists := h.rows(); lists != nil {
		rows, err = lists.ListRows(pluginID, request.Msg.GetOperation())
	}
	if err != nil {
		return nil, wire.NewRowListError(ctx, err)
	}
	return connect.NewResponse(&v1.ConnectionCallServiceListRowsResponse{Rows: wire.ConvertCallRowsToWire(rows)}), nil
}

// ListRowOptions lists the values a row field of a connection among the rows of the plugin can take. A host
// with no rows answers not found.
func (h *Handler) ListRowOptions(
	ctx context.Context, request *connect.Request[v1.ConnectionCallServiceListRowOptionsRequest],
) (*connect.Response[v1.ConnectionCallServiceListRowOptionsResponse], error) {
	ctx, pluginID, err := h.identify(ctx, request.Header(), request.Msg.GetPluginId())
	if err != nil {
		return nil, err
	}
	lists := h.rows()
	if lists == nil {
		return nil, wire.NewRowListError(ctx, connection.ErrNotFound)
	}
	message := request.Msg
	options, err := lists.ListRowOptions(
		ctx, pluginID, message.GetOperation(), message.GetConnectionId(), message.GetField(), message.GetOptions(),
	)
	if err != nil {
		return nil, wire.NewRowListError(ctx, err)
	}
	return connect.NewResponse(&v1.ConnectionCallServiceListRowOptionsResponse{Options: wire.ConvertOptionsToWire(options)}), nil
}

func getCancelKey(pluginID, callID string) string {
	if callID == "" {
		return ""
	}
	return pluginID + "\n" + callID
}
