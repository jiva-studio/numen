// Package connectioncall serves a window's ConnectionCallService: the operations a plugin view asks for,
// through the rows of the plugin the request names.
package connectioncall

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

var errPluginToken = errors.New("ConnectionCallService is not served to a plugin")

// Calls carries out the operations plugins ask for.
type Calls interface {
	Call(ctx context.Context, callerPluginID string, request modelcall.Request, send modelcall.Send) ([]byte, error)
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
		Operation: request.Msg.GetOperation(), Payload: request.Msg.GetPayload(), Priority: readPriority(request.Msg.GetPriority()),
	}, func(data []byte) error {
		return stream.Send(&v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Data{Data: data}})
	})
	if err == nil {
		return stream.Send(&v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Done{Done: done}})
	}
	failure, isServed := writeCallFailure(ctx, err)
	if !isServed {
		return newStatusError(ctx, err)
	}
	return stream.Send(&v1.ConnectionCallServiceCallResponse{Frame: &v1.ConnectionCallServiceCallResponse_Failure{Failure: failure}})
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

func newStatusError(ctx context.Context, err error) error {
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
