// Package completion answers the completion and chat requests of a connection: it checks the level of
// reasoning, gives the request the time it may take and asks the service.
package completion

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
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

// Complete asks the service for an answer. A level of reasoning outside those the model supports is a stop.
func (c *Completion) Complete(ctx context.Context, call connection.Call, request *llm.CompleteRequest) (*llm.CompleteResult, error) {
	model := call.Options[llm.OptionModel]
	if !reasoning.IsSupported(call.Type, model, call.Options[llm.OptionReasoning]) {
		return nil, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	address := call.Fields[servicetype.FieldBaseURL]
	result, err := c.gateway.Complete(ctx, call, c.limits.Get(call.Type, address, model, operation.Complete), request)
	if err != nil {
		return nil, err
	}
	c.limits.MarkServed(address, model)
	return result, nil
}

// Chat asks the service for the next turn and sends its frames as they arrive. A level of reasoning outside
// the levels the model supports is a stop.
func (c *Completion) Chat(
	ctx context.Context, call connection.Call, request *llm.ChatRequest, send func(*llm.ChatFrame) error,
) (*llm.ChatFinish, error) {
	model := call.Options[llm.OptionModel]
	if !reasoning.IsSupported(call.Type, model, call.Options[llm.OptionReasoning]) {
		return nil, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	address := call.Fields[servicetype.FieldBaseURL]
	finish, err := c.gateway.Chat(ctx, call, c.limits.Get(call.Type, address, model, operation.Chat), request, send)
	if err != nil {
		return nil, err
	}
	c.limits.MarkServed(address, model)
	return finish, nil
}
