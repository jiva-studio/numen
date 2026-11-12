// Package conversation holds what an agent and a model exchange: messages, tool calls and the tools
// offered to the model.
package conversation

import "encoding/json"

// Message is one message of a conversation. Role is system, user, assistant or tool. An assistant
// message carries the calls it asked for, and a tool message carries the id of the call it answers.
type Message struct {
	Role       string
	Text       string
	ToolCalls  []ToolCall
	ToolCallID string
}

// ToolCall is a call of a tool the model asked for. Arguments is one JSON object.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// Tool is a tool of the host offered to the model. Parameters is the JSON schema of its arguments, and
// Instructions are the sentences the tool adds to the instructions of the model.
type Tool struct {
	Name         string
	Title        string
	Description  string
	Parameters   json.RawMessage
	IsReadOnly   bool
	Instructions []string
}

// Scope is what every tool call of a task carries: the context of the task and the vault the tools work.
// An empty VaultID is the vault the window shows.
type Scope struct {
	Context map[string]string
	VaultID string
}

// ToolResult is the result of a tool call as text. IsError is set for a call that failed.
type ToolResult struct {
	Text    string
	IsError bool
}

// Turn is what the model is asked in one request.
type Turn struct {
	Messages    []Message
	Tools       []Tool
	Temperature *float64
	MaxTokens   int
}

// Reply is the model's answer to a turn. IsCut is set for an answer cut off at the model's length limit.
type Reply struct {
	Message Message
	IsCut   bool
}
