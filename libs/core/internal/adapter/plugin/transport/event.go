package transport

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

const eventTimeout = 3 * time.Second

var ErrEventRefused = errors.New("plugin rejected the event")

// DeliverPluginEvent hands an event to a running plugin and returns errEventRefused when the plugin
// answers with an error. A plugin that does not serve numen.v1.PluginEventService does not take
// events, and delivery to it succeeds.
func DeliverPluginEvent(ctx context.Context, endpoint Endpoint, event string) error {
	client := numenv1connect.NewPluginEventServiceClient(
		EndpointClient(endpoint),
		endpoint.BaseURL(),
		connect.WithInterceptors(AuthInterceptor(endpoint.Token)),
	)
	eventCtx, cancel := context.WithTimeout(ctx, eventTimeout)
	defer cancel()
	resp, err := client.DeliverEvent(eventCtx, connect.NewRequest(&v1.DeliverEventRequest{Event: event}))
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnimplemented {
			return nil
		}
		return err
	}
	return ConvertEventRefusal(resp.Msg.GetError(), resp.Msg.GetReason())
}

// ConvertEventRefusal returns the error a plugin's answer to an event stands for, nil when it took the event.
func ConvertEventRefusal(code v1.PluginEventErrorCode, reason string) error {
	switch code {
	case v1.PluginEventErrorCode_PLUGIN_EVENT_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.PluginEventErrorCode_PLUGIN_EVENT_ERROR_CODE_REFUSED:
		return fmt.Errorf("%w: %s", ErrEventRefused, reason)
	}
	return fmt.Errorf("%w with unknown code %d: %s", ErrEventRefused, code, reason)
}
