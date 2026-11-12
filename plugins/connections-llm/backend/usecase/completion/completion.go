// Package completion answers the completion and chat requests of a connection: it checks the level of
// reasoning, gives the request the time it may take and asks the service.
package completion

import (
	"context"
	"errors"
	"time"

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
	asked, err := c.getTarget(call, operation.Complete)
	if err != nil {
		return nil, err
	}
	result, err := c.gateway.Complete(ctx, call, asked.limit, request)
	if err != nil {
		return nil, err
	}
	c.limits.MarkServed(asked.address, asked.model)
	return result, nil
}

// Chat asks the service for the next turn and sends its frames as they arrive. A level of reasoning outside
// the levels the model supports is a stop.
func (c *Completion) Chat(
	ctx context.Context, call connection.Call, request *llm.ChatRequest, send func(*llm.ChatFrame) error,
) (*llm.ChatFinish, error) {
	asked, err := c.getTarget(call, operation.Chat)
	if err != nil {
		return nil, err
	}
	finish, err := c.gateway.Chat(ctx, call, asked.limit, request, send)
	if err != nil {
		return nil, err
	}
	c.limits.MarkServed(asked.address, asked.model)
	return finish, nil
}

// target is where a request goes: the model, the address of the service and the time the request may take.
type target struct {
	model   string
	address string
	limit   time.Duration
}

func (c *Completion) getTarget(call connection.Call, op string) (target, error) {
	model := call.Options[llm.OptionModel]
	if !reasoning.IsSupported(call.Type, model, call.Options[llm.OptionReasoning]) {
		return target{model: "", address: "", limit: 0}, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	address := call.Fields[servicetype.FieldBaseURL]
	return target{model: model, address: address, limit: c.limits.Get(call.Type, address, model, op)}, nil
}
