package task

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

// Tools lists the tools of the host the plugin may offer and runs a call of one.
type Tools interface {
	ListTools(ctx context.Context) (tools.Listing, error)
	InvokeTool(ctx context.Context, call tools.Call) (*tools.CallResponse, error)
}

// Invoke runs one call of a served tool with the JSON arguments of the agent.
type Invoke func(ctx context.Context, name string, arguments []byte) (*tools.CallResponse, error)

// ToolServer serves tools to an agent program.
type ToolServer interface {
	Serve(ctx context.Context, listed []*tools.ToolInfo, invoke Invoke) (ServedTools, error)
}

// ServedTools is where tools are served, the token presented there, and how the serving ends.
type ServedTools interface {
	GetURL() string
	GetToken() string
	Close(ctx context.Context) error
}

// Reasoning chooses the depth of reasoning of a run.
type Reasoning interface {
	GetReasoning(model, requested string) string
}

// TempDirs creates the folder the runs start in and removes it.
type TempDirs interface {
	Create() (string, error)
	Remove(dir string) error
}
