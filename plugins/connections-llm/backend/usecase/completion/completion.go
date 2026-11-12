// Package completion answers the completion and chat requests of a connection: it checks the level of
// reasoning, gives the request the time it may take and asks the service.
package completion

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/operation"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/reasoning"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/ports"
)

// Completion carries out the complete and chat operations.
type Completion struct {
	gateway ports.Gateway
	limits  *limits.Limits
}

// New returns a Completion that asks gateway and takes the time of a request from limits.
func New(gateway ports.Gateway, requestLimits *limits.Limits) (*Completion, error) {
	if gateway == nil || requestLimits == nil {
		return nil, errors.New("completion: a gateway and limits are required")
	}
	return &Completion{gateway: gateway, limits: requestLimits}, nil
}

// Complete asks the service for an answer. A level of reasoning the model does not take is a stop.
func (c *Completion) Complete(ctx context.Context, call connection.Call, request connection.ProviderCompleteRequest) (connection.CompleteResult, error) {
	if !reasoning.IsSupported(call.Type, call.Model, call.Reasoning) {
		return connection.CompleteResult{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	address := call.Fields[servicetype.FieldBaseURL]
	result, err := c.gateway.Complete(ctx, call, c.limits.Get(call.Type, address, call.Model, operation.Complete), request)
	if err != nil {
		return connection.CompleteResult{}, err
	}
	c.limits.MarkServed(address, call.Model)
	return result, nil
}

// Chat asks the service for the next turn and sends its events as they arrive. A level of reasoning the
// model does not take is a stop.
func (c *Completion) Chat(ctx context.Context, call connection.Call, request connection.ProviderChatRequest, send func(connection.ChatEvent) error) error {
	if !reasoning.IsSupported(call.Type, call.Model, call.Reasoning) {
		return connection.NewStopError(connection.ReasonBadRequest)
	}
	address := call.Fields[servicetype.FieldBaseURL]
	if err := c.gateway.Chat(ctx, call, c.limits.Get(call.Type, address, call.Model, operation.Chat), request, send); err != nil {
		return err
	}
	c.limits.MarkServed(address, call.Model)
	return nil
}
