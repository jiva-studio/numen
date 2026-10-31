package agentrun

import (
	"context"
	"sync"
)

// Toolbox keeps the tools and the system prompt extension the host registers with an agent plugin.
// Embedded in the plugin's agent, it answers RegisterTool, ExtendSystemPrompt and GetTools. The zero
// value is ready.
type Toolbox struct {
	mu           sync.RWMutex
	systemPrompt string
	tools        []*ToolDefinition
}

// RegisterTool keeps the tool.
func (b *Toolbox) RegisterTool(_ context.Context, tool *ToolDefinition) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if tool != nil {
		b.tools = append(b.tools, tool)
	}
	return nil
}

// ExtendSystemPrompt appends the extension to the system prompt on a line of its own and returns
// the whole prompt.
func (b *Toolbox) ExtendSystemPrompt(_ context.Context, extension string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if extension != "" {
		if b.systemPrompt != "" {
			b.systemPrompt += "\n" + extension
		} else {
			b.systemPrompt = extension
		}
	}
	return b.systemPrompt, nil
}

// GetTools returns the registered tools and the system prompt extension.
func (b *Toolbox) GetTools(context.Context) (*ToolSet, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return &ToolSet{
		Tools:                 b.tools,
		SystemPromptExtension: b.systemPrompt,
	}, nil
}
