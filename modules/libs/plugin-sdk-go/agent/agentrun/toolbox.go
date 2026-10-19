package agentrun

import (
	"context"
	"sync"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// Toolbox keeps the tools and the system prompt extension the host registers with an agent plugin.
// Embedded in the plugin's agent server, it answers RegisterTool, ExtendSystemPrompt and
// GetAgentTools. The zero value is ready.
type Toolbox struct {
	mu           sync.RWMutex
	systemPrompt string
	tools        []*numenv1.AgentToolDefinition
}

// RegisterTool keeps the tool of the request.
func (b *Toolbox) RegisterTool(
	_ context.Context,
	req *connect.Request[numenv1.RegisterToolRequest],
) (*connect.Response[numenv1.RegisterToolResponse], error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if tool := req.Msg.GetTool(); tool != nil {
		b.tools = append(b.tools, tool)
	}
	return connect.NewResponse(&numenv1.RegisterToolResponse{Success: true}), nil
}

// ExtendSystemPrompt appends the extension to the system prompt on a line of its own.
func (b *Toolbox) ExtendSystemPrompt(
	_ context.Context,
	req *connect.Request[numenv1.ExtendSystemPromptRequest],
) (*connect.Response[numenv1.ExtendSystemPromptResponse], error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ext := req.Msg.GetExtension(); ext != "" {
		if b.systemPrompt != "" {
			b.systemPrompt += "\n" + ext
		} else {
			b.systemPrompt = ext
		}
	}
	return connect.NewResponse(&numenv1.ExtendSystemPromptResponse{
		FullPrompt: b.systemPrompt,
	}), nil
}

// GetAgentTools returns the registered tools and the system prompt extension.
func (b *Toolbox) GetAgentTools(
	_ context.Context,
	_ *connect.Request[numenv1.GetAgentToolsRequest],
) (*connect.Response[numenv1.GetAgentToolsResponse], error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return connect.NewResponse(&numenv1.GetAgentToolsResponse{
		Tools:                 b.tools,
		SystemPromptExtension: b.systemPrompt,
	}), nil
}
