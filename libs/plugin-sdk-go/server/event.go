package server

import (
	"context"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// EventHandler receives an event the plugin's manifest subscribes to in activationEvents. An error it
type EventHandler func(ctx context.Context, event string) error

// WithEventHandler mounts numen.v1.PluginEventService, which hands each delivered event to fn.
// A server without one does not take events.
func WithEventHandler(fn EventHandler) Option {
	return func(s *Server) {
		s.onEvent = fn
	}
}

// eventServer implements numen.v1.PluginEventService.
type eventServer struct {
	numenv1connect.UnimplementedPluginEventServiceHandler
	handle EventHandler
}

// that carries its text; the call itself succeeds.
func (s eventServer) DeliverEvent(
	ctx context.Context,
	req *connect.Request[numenv1.DeliverEventRequest],
) (*connect.Response[numenv1.DeliverEventResponse], error) {
	answer := &numenv1.DeliverEventResponse{}
	if err := s.handle(ctx, req.Msg.GetEvent()); err != nil {
		answer.Error = numenv1.PluginEventErrorCode_PLUGIN_EVENT_ERROR_CODE_REFUSED
		answer.Reason = err.Error()
	}
	return connect.NewResponse(answer), nil
}
