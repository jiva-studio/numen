// Package toolset keeps the tools and the system prompt extension the host registers with the agent.
package toolset

import (
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

// Toolset holds what the host registered; the zero value holds nothing.
type Toolset struct {
	mu     sync.RWMutex
	tools  []*agentrun.ToolDefinition
	prompt string
}

// Register keeps the tool.
func (t *Toolset) Register(tool *agentrun.ToolDefinition) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tools = append(t.tools, tool)
}

// ExtendPrompt appends the extension to the system prompt and returns the whole prompt.
func (t *Toolset) ExtendPrompt(extension string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prompt += extension
	return t.prompt
}

// GetPrompt returns the system prompt.
func (t *Toolset) GetPrompt() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.prompt
}

// Get returns a copy of the registered tools and the system prompt.
func (t *Toolset) Get() *agentrun.ToolSet {
	t.mu.RLock()
	defer t.mu.RUnlock()
	toolsCopy := make([]*agentrun.ToolDefinition, len(t.tools))
	copy(toolsCopy, t.tools)
	return &agentrun.ToolSet{Tools: toolsCopy, SystemPromptExtension: t.prompt}
}
