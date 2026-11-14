// Package ports declares what the use cases of the agent require from the models, the tools of the host
// and the clock, in the words of those use cases.
package ports

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
)

// Chat asks the model of the turn for the next message of a conversation. Stream passes the reply text to
// emit as it arrives and returns the first error of emit.
type Chat interface {
	Stream(ctx context.Context, turn conversation.Turn, emit func(string) error) (conversation.Reply, error)
}

// Models are the connections that serve llm.chat and the models each lists.
type Models interface {
	ListConnections(ctx context.Context) ([]conversation.ChatConnection, error)
	ListModels(ctx context.Context, connection conversation.ChatConnection) ([]conversation.ModelOption, error)
}

// Tools are the tools of the host the plugin may call.
type Tools interface {
	ListTools(ctx context.Context) ([]conversation.Tool, error)
	CallTool(ctx context.Context, name, arguments string, scope conversation.Scope) conversation.ToolResult
}

// Clock tells the time a conversation was last used.
type Clock interface {
	Now() time.Time
}
