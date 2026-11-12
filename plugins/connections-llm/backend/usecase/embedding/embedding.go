// Package embedding carries out the embed operation of a connection, reads what is known of the profile
// of a model, and measures the width of its vectors.
package embedding

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/embeddingmodel"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/operation"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/ports"
)

const probeText = "width probe"

// Embedding carries out the embed operation.
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
func (e *Embedding) Embed(ctx context.Context, call connection.Call, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ctx.Err()
	}
	address := call.Fields[servicetype.FieldBaseURL]
	vectors, err := e.gateway.Embed(ctx, call, e.limits.Get(call.Type, address, call.Model, operation.Embed), texts)
	if err != nil {
		return nil, err
	}
	e.limits.MarkServed(address, call.Model)
	return vectors, nil
}

// ProbeWidth embeds one short text and returns the length of its vector.
func (e *Embedding) ProbeWidth(ctx context.Context, call connection.Call) (int, error) {
	vectors, err := e.Embed(ctx, call, []string{probeText})
	if err != nil {
		return 0, err
	}
	return len(vectors[0]), nil
}

// GetProfile returns what the plugin knows of an embedding model from its table, with no request.
// A model that is not in the table has an unknown profile.
func (e *Embedding) GetProfile(_ context.Context, call connection.Call) (connection.ModelProfile, error) {
	if !servicetype.IsKnown(call.Type) {
		return connection.ModelProfile{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	base, err := address.Parse(call.Fields[servicetype.FieldBaseURL], false)
	if err != nil {
		return connection.ModelProfile{}, connection.NewStopError(connection.ReasonBadRequest)
	}
	profile := connection.ModelProfile{
		Source: servicetype.PluginID + "/" + call.Type + " " + base.Scheme + "://" + base.Host + strings.TrimSuffix(base.Path, "/"),
	}
	if known, ok := embeddingmodel.Find(call.Model); ok {
		profile.IsKnown, profile.Dimensions, profile.MaxTokens, profile.Pooling = true, known.Dimensions, known.MaxTokens, known.Pooling
	}
	return profile, nil
}
