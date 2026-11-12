// Package models lists the models and the reasoning levels a connection offers.
package models

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

// Models lists what a connection offers.
type Models struct {
	gateway ports.Gateway
	limits  *limits.Limits
}

// New returns Models that ask gateway and take the time of a request from limits.
func New(gateway ports.Gateway, requestLimits *limits.Limits) (*Models, error) {
	if gateway == nil || requestLimits == nil {
		return nil, errors.New("models: a gateway and limits are required")
	}
	return &Models{gateway: gateway, limits: requestLimits}, nil
}

// ListModels lists the models of the service that fit the operations of a setting of rows.
func (m *Models) ListModels(ctx context.Context, call connection.Call, operations []string) ([]connection.Option, error) {
	limit := m.limits.Get(call.Type, call.Fields[servicetype.FieldBaseURL], call.Options[llm.OptionModel], operation.List)
	return m.gateway.ListModels(ctx, call, limit, operations)
}

// ListReasoningLevels returns the levels the model of the row takes, or none.
func (m *Models) ListReasoningLevels(_ context.Context, call connection.Call) ([]connection.Option, error) {
	levels := reasoning.GetLevels(call.Type, call.Options[llm.OptionModel])
	out := make([]connection.Option, len(levels))
	for i, one := range levels {
		out[i] = connection.Option{Value: one.Value, Title: one.Title}
	}
	return out, nil
}
