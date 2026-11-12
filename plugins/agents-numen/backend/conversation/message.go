// Package conversation holds what an agent and a model exchange: messages, tool calls and the tools
// offered to the model.
package conversation

import (
	"encoding/json"
	"errors"
)

// ErrNoConnection is a request that no connection could answer because the agent has none.
var ErrNoConnection = errors.New("the agent has no connection")

// ReasonNoConnection is why a run stops, and why the agent is unreachable, while no row names a connection.
const ReasonNoConnection = "the Numen Agent has no connection: choose one in its settings"

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

// Tool is a tool offered to the model; Parameters is the JSON schema of its arguments.
type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage
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

// ToolServer is where the tools of a task are served, and the part of the task's context sent
// with each call. It holds no address when the task names no tool.
type ToolServer struct {
	URL   string
	Token string
	Names []string
	Meta  map[string]string
}
