package modelcall

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// Complete asks the rows of the plugin for the operation complete, in order. It returns
// connection.ErrNoConnection where the plugin has no row, *connection.AllFailedError where every
// row failed, *connection.RejectedError where a connection found the request wrong, and the error of
// the context where the call was canceled.
func (c *Calls) Complete(
	ctx context.Context, callerPluginID string, request modelcall.CompleteRequest, priority modelcall.Priority,
) (modelcall.CompleteResult, error) {
	return carry(ctx, c, asking{callerPluginID, "complete", priority}, pickCompletion, func(
		ctx context.Context, provider connectionport.CompletionProvider, call modelcall.ProviderCall, _ connection.Row,
	) (modelcall.CompleteResult, error) {
		return provider.Complete(ctx, call, request)
	})
}

// Embed asks the rows of the plugin for the operation embed, in order, and returns one vector for each
// text. It returns the errors Complete does. No text is no call.
func (c *Calls) Embed(
	ctx context.Context, callerPluginID string, texts []string, priority modelcall.Priority,
) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	return carry(ctx, c, asking{callerPluginID, "embed", priority}, pickEmbedding, func(
		ctx context.Context, provider connectionport.EmbeddingProvider, call modelcall.ProviderCall, _ connection.Row,
	) ([][]float32, error) {
		return provider.Embed(ctx, call, texts)
	})
}

// GetEmbeddingProfile asks the rows of the plugin for the operation embed, in order, for the profile of
// their model. Where the provider has no profile it measures the width with one probe. It returns the
// errors Complete does.
func (c *Calls) GetEmbeddingProfile(ctx context.Context, callerPluginID string) (modelcall.EmbeddingProfile, error) {
	return carry(ctx, c, asking{callerPluginID, "embed", modelcall.PriorityBackground}, pickProfiles, func(
		ctx context.Context, provider connectionport.ModelProfiles, call modelcall.ProviderCall, row connection.Row,
	) (modelcall.EmbeddingProfile, error) {
		profile, err := provider.GetModelProfile(ctx, call)
		if err != nil {
			return modelcall.EmbeddingProfile{}, err
		}
		if !profile.IsKnown {
			profile.Dimensions, err = provider.ProbeEmbeddingWidth(ctx, call)
			if err != nil {
				return modelcall.EmbeddingProfile{}, err
			}
		}
		return modelcall.EmbeddingProfile{
			Model: row.Model, Source: profile.Source, Pooling: profile.Pooling, QueryPrefix: profile.QueryPrefix,
			DocumentPrefix: profile.DocumentPrefix, Dimensions: profile.Dimensions, MaxTokens: profile.MaxTokens,
		}, nil
	})
}
