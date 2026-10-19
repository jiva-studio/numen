package server

import (
	"context"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// EventHandler receives an event the plugin's manifest subscribes to in activationEvents.
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

// DeliverEvent passes the event to the plugin's handler and answers its error.
func (s eventServer) DeliverEvent(
	ctx context.Context,
	req *connect.Request[numenv1.DeliverEventRequest],
) (*connect.Response[numenv1.DeliverEventResponse], error) {
	if err := s.handle(ctx, req.Msg.GetEvent()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&numenv1.DeliverEventResponse{}), nil
}
