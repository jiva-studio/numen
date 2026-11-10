// Package conversation holds what an agent and a chat API exchange: messages, tool calls, the
// tools offered to a model, and the reasoning depths a model takes.
package conversation

import (
	"encoding/json"
	"errors"
)

// FinishLength is the reason a chat API gives for a reply it cut off at the model's length limit.
const FinishLength = "length"

// ErrStreamRejected is a streaming request the chat API refused, which a request for the whole
// reply may still succeed.
var ErrStreamRejected = errors.New("the chat API rejected a streaming request")

// Message is one message of a conversation with the chat API.
type Message struct {
	Role       string     `json:"role"`
	Content    *string    `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// CallFunction names the function a model calls and the JSON arguments it passes.
type CallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCall is a function call requested by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function CallFunction `json:"function"`
}

// ToolFunction describes a function offered to the model; Parameters is its JSON schema.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

// FunctionTool is a function offered to the model.
type FunctionTool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// Turn is what the model is asked in one request.
type Turn struct {
	Model    string
	Effort   Effort
	Messages []Message
	Tools    []FunctionTool
}

// Reply is the model's answer to a turn and the reason the chat API gave for ending it.
type Reply struct {
	Message      Message
	FinishReason string
}

// ToolServer is where the tools of a task are served, and the part of the task's context sent
// with each call. It holds no address when the task names no tool.
type ToolServer struct {
	URL   string
	Token string
	Names []string
	Meta  map[string]string
}
