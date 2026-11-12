// Package ports declares what the use cases of the agent require from the model, the tools of the host
// and the clock, in the words of those use cases.
package ports

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
)

// Chat asks the model for the next message of a conversation. Stream passes the reply text to emit as it
// arrives and returns the first error of emit.
type Chat interface {
	Stream(ctx context.Context, turn conversation.Turn, emit func(string) error) (conversation.Reply, error)
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
