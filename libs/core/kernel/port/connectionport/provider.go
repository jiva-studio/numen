package connectionport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// CompletionProvider carries out the operation complete. A call fails with *connection.TryNextError or
// *connection.StopError, with the error of the context when it was canceled, or with an error of its own
// when the provider could not be reached.
type CompletionProvider interface {
	Complete(ctx context.Context, call modelcall.ProviderCall, request modelcall.CompleteRequest) (modelcall.CompleteResult, error)
}

// EmbeddingProvider carries out the operation embed and returns the vectors in the order of the texts.
type EmbeddingProvider interface {
	Embed(ctx context.Context, call modelcall.ProviderCall, texts []string) ([][]float32, error)
}

// ChatProvider carries out the operation chat. It sends the events of one turn in order and ends with
// one finish. It returns the error of the first send that fails.
type ChatProvider interface {
	Chat(ctx context.Context, call modelcall.ProviderCall, request modelcall.ChatRequest, send func(modelcall.ChatEvent) error) error
}

// ModelCatalog lists the models of a connection and the reasoning levels of the model of the call.
type ModelCatalog interface {
	ListModels(ctx context.Context, call modelcall.ProviderCall, operation string) ([]modelcall.Model, error)
	ListReasoningLevels(ctx context.Context, call modelcall.ProviderCall) ([]modelcall.ReasoningLevel, error)
}

// ModelProfiles reads the profile of an embedding model and measures the width of its vectors.
type ModelProfiles interface {
	GetModelProfile(ctx context.Context, call modelcall.ProviderCall) (modelcall.ModelProfile, error)
	ProbeEmbeddingWidth(ctx context.Context, call modelcall.ProviderCall) (int, error)
}

// Provider is what a plugin that provides connection types carries out for the calls of other plugins.
type Provider interface {
	CompletionProvider
	ChatProvider
	EmbeddingProvider
	ModelProfiles
}

// ProviderSource opens the provider of a plugin.
type ProviderSource interface {
	// OpenProvider starts the plugin if it is not running and returns its provider with the function that
	// releases it. A plugin that cannot be started is an error.
	OpenProvider(ctx context.Context, pluginID string) (Provider, func(), error)
}

// CatalogSource opens the model catalog of a plugin.
type CatalogSource interface {
	// OpenCatalog starts the plugin if it is not running and returns its catalog with the function that
	// releases it. A plugin that cannot be started is an error.
	OpenCatalog(ctx context.Context, pluginID string) (ModelCatalog, func(), error)
}
