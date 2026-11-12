// Package ports declares what the use cases of the agent require from the model and the vault's
// tools, in the words of those use cases.
package ports

import (
	"context"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
)

// Chat asks the model for the next message of a conversation. Stream passes the reply text to emit as
// it arrives. An error that wraps conversation.ErrNoConnection means the agent has no connection;
// any other error carries the reason the connections gave.
type Chat interface {
	Stream(ctx context.Context, turn conversation.Turn, emit func(string) error) (conversation.Reply, error)
}

// Toolbox is the vault's tools as one run sees them: the tools it may offer the model, and the call
// of one.
type Toolbox interface {
	Tools() []conversation.Tool
	// Call returns the result of the tool as text, or a text that starts with "error:".
	Call(ctx context.Context, name, arguments string) string
	Close() error
}

// ToolOpener connects to the tool server of a task. The toolbox it returns offers nothing where the
// task names no tool.
type ToolOpener interface {
	OpenTools(ctx context.Context, server conversation.ToolServer) (Toolbox, error)
}
