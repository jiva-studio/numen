package plugin

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

const eventTimeout = 3 * time.Second

// deliverPluginEvent hands an event to a running plugin. A plugin that does not serve
// numen.v1.PluginEventService does not take events, and delivery to it succeeds.
func deliverPluginEvent(ctx context.Context, endpoint Endpoint, event string) error {
	client := numenv1connect.NewPluginEventServiceClient(
		EndpointClient(endpoint),
		endpoint.BaseURL(),
		connect.WithInterceptors(authInterceptor(endpoint.Token)),
	)
	eventCtx, cancel := context.WithTimeout(ctx, eventTimeout)
	defer cancel()
	_, err := client.DeliverEvent(eventCtx, connect.NewRequest(&v1.DeliverEventRequest{Event: event}))
	var connectErr *connect.Error
	if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnimplemented {
		return nil
	}
	return err
}
