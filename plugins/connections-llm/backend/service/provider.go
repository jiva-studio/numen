package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/completion"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/embedding"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/models"
)

// provider answers the calls of the provider service with the use cases of the plugin.
type provider struct {
	completion *completion.Completion
	embedding  *embedding.Embedding
	models     *models.Models
}

func (p *provider) roles() connection.Provider {
	return connection.Provider{Completer: p, Chatter: p, Embedder: p, Models: p, Profiles: p}
}

func (p *provider) Complete(ctx context.Context, call connection.Call, request connection.ProviderCompleteRequest) (connection.CompleteResult, error) {
	return p.completion.Complete(ctx, call, request)
}

func (p *provider) Chat(ctx context.Context, call connection.Call, request connection.ProviderChatRequest, send func(connection.ChatEvent) error) error {
	return p.completion.Chat(ctx, call, request, send)
}

func (p *provider) Embed(ctx context.Context, call connection.Call, texts []string) ([][]float32, error) {
	return p.embedding.Embed(ctx, call, texts)
}

func (p *provider) ProbeEmbeddingWidth(ctx context.Context, call connection.Call) (int, error) {
	return p.embedding.ProbeWidth(ctx, call)
}

func (p *provider) GetModelProfile(ctx context.Context, call connection.Call) (connection.ModelProfile, error) {
	return p.embedding.GetProfile(ctx, call)
}

func (p *provider) ListModels(ctx context.Context, call connection.Call, operation string) ([]connection.Model, error) {
	return p.models.ListModels(ctx, call, operation)
}

func (p *provider) ListReasoningLevels(ctx context.Context, call connection.Call) ([]connection.ReasoningLevel, error) {
	return p.models.ListReasoningLevels(ctx, call)
}
