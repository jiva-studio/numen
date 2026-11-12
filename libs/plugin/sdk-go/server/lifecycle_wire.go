package server

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// lifecycleHandler serves a LifecycleServer as numen.v1.PluginLifecycleService.
type lifecycleHandler struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler

	state *LifecycleServer
}

func (h lifecycleHandler) Ping(
	context.Context, *connect.Request[v1.PingRequest],
) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(writePing(PingResult{Status: statusOK, Protocol: ProtocolVersion})), nil
}

func (h lifecycleHandler) Shutdown(
	context.Context, *connect.Request[v1.ShutdownRequest],
) (*connect.Response[v1.ShutdownResponse], error) {
	if err := h.state.shutdown(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ShutdownResponse{Success: true}), nil
}

func (h lifecycleHandler) Configure(
	ctx context.Context, req *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	if err := h.state.configureWith(ctx, req.Msg.GetValues()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ConfigureResponse{Success: true}), nil
}

// PingResult is the answer of a plugin to a ping.
//
//numen:api
type PingResult struct {
	// Status is "ok" for a healthy plugin.
	Status string
	// Protocol is the protocol version the plugin uses.
	Protocol int32
}

func writePing(result PingResult) *v1.PingResponse {
	return &v1.PingResponse{Status: result.Status, Protocol: result.Protocol}
}
