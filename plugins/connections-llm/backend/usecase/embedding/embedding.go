// Package embedding carries out the embed operation of a connection and reads the profile of the
// embedding model of a row: from the table where the model is in it, else by measuring the width of
// its vectors.
package embedding

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/embeddingmodel"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/operation"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/ports"
)

const probeText = "width probe"

// Embedding carries out the embed and profile operations.
type Embedding struct {
	gateway ports.Gateway
	limits  *limits.Limits
}

// New returns an Embedding that asks gateway and takes the time of a request from limits.
func New(gateway ports.Gateway, requestLimits *limits.Limits) (*Embedding, error) {
	if gateway == nil || requestLimits == nil {
		return nil, errors.New("embedding: a gateway and limits are required")
	}
	return &Embedding{gateway: gateway, limits: requestLimits}, nil
}

// Embed asks the service for one vector for each text.
func (e *Embedding) Embed(ctx context.Context, call connection.Call, request *llm.EmbedRequest) (*llm.EmbedResult, error) {
	vectors, err := e.embedTexts(ctx, call, request.GetTexts())
	if err != nil {
		return nil, err
	}
	result := &llm.EmbedResult{Vectors: make([]*llm.Vector, len(vectors))}
	for i, one := range vectors {
		result.Vectors[i] = &llm.Vector{Values: one}
	}
	return result, nil
}

func (e *Embedding) embedTexts(ctx context.Context, call connection.Call, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ctx.Err()
	}
	model := call.Options[llm.OptionModel]
	address := call.Fields[servicetype.FieldBaseURL]
	vectors, err := e.gateway.Embed(ctx, call, e.limits.Get(call.Type, address, model, operation.Embed), texts)
	if err != nil {
		return nil, err
	}
	e.limits.MarkServed(address, model)
	return vectors, nil
}

// GetProfile returns the profile of the embedding model of the row: the table gives it with no request
// for a model it holds, and the width of any other model is measured by embedding one short text.
func (e *Embedding) GetProfile(ctx context.Context, call connection.Call) (*llm.Profile, error) {
	if !servicetype.IsKnown(call.Type) {
		return nil, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	base, err := address.Parse(call.Fields[servicetype.FieldBaseURL], false)
	if err != nil {
		return nil, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	model := call.Options[llm.OptionModel]
	profile := &llm.Profile{
		Model:  model,
		Source: servicetype.PluginID + "/" + call.Type + " " + base.Scheme + "://" + base.Host + strings.TrimSuffix(base.Path, "/"),
	}
	if known, ok := embeddingmodel.Find(model); ok {
		profile.Width, profile.MaxTokens, profile.Pooling = int32(known.Dimensions), int32(known.MaxTokens), known.Pooling
		return profile, nil
	}
	vectors, err := e.embedTexts(ctx, call, []string{probeText})
	if err != nil {
		return nil, err
	}
	profile.Width = int32(len(vectors[0]))
	return profile, nil
}
