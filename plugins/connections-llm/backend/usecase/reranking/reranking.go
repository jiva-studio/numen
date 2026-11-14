// Package reranking carries out the rerank operation of a connection.
package reranking

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/operation"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/ports"
)

// Reranking carries out the rerank operation.
type Reranking struct {
	gateway ports.Gateway
	limits  *limits.Limits
}

// New returns a Reranking that asks gateway and takes the time of a request from limits.
func New(gateway ports.Gateway, requestLimits *limits.Limits) (*Reranking, error) {
	if gateway == nil || requestLimits == nil {
		return nil, errors.New("reranking: a gateway and limits are required")
	}
	return &Reranking{gateway: gateway, limits: requestLimits}, nil
}

// Rerank asks the service for one score for each document. No document is no request.
func (r *Reranking) Rerank(ctx context.Context, call connection.Call, request *llm.RerankRequest) (*llm.RerankResult, error) {
	if len(request.GetDocuments()) == 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return &llm.RerankResult{}, nil
	}
	model := call.Options[llm.OptionModel]
	address := call.Fields[servicetype.FieldBaseURL]
	scores, err := r.gateway.Rerank(ctx, call, r.limits.Get(call.Type, address, model, operation.Rerank), request)
	if err != nil {
		return nil, err
	}
	r.limits.MarkServed(address, model)
	return &llm.RerankResult{Scores: scores}, nil
}
