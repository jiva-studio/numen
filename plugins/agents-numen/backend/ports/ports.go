// Package ports declares what the use cases of the agent require from the chat API, the vault's
// tools and the catalog of models, in the words of those use cases.
package ports

import (
	"context"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
)

// Chat asks a model for the next message of a conversation. Stream passes the reply text to emit as it
// arrives; an error that wraps conversation.ErrStreamRejected means Send may still succeed.
type Chat interface {
	Stream(ctx context.Context, cfg settings.Settings, turn conversation.Turn, emit func(string) error) (conversation.Reply, error)
	Send(ctx context.Context, cfg settings.Settings, turn conversation.Turn) (conversation.Reply, error)
}

// ModelLister lists the models the chat API of the settings offers.
type ModelLister interface {
	ListModels(ctx context.Context, cfg settings.Settings) ([]conversation.Model, error)
}

// ModelCatalog lists the models of the configured chat API.
type ModelCatalog interface {
	ListModels(ctx context.Context) ([]conversation.Model, error)
}

// Toolbox is the vault's tools as one run sees them: the tools it may offer the model, and the call
// of one.
type Toolbox interface {
	Tools() []conversation.FunctionTool
	// Call returns the result of the tool as text, or a text that starts with "error:".
	Call(ctx context.Context, name, arguments string) string
	Close() error
}

// ToolOpener connects to the tool server of a task. The toolbox it returns offers nothing where the
// task names no tool.
type ToolOpener interface {
	OpenTools(ctx context.Context, server conversation.ToolServer) (Toolbox, error)
}
