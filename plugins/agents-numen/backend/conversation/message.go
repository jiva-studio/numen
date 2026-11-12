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
	Role       string
	Content    *string
	ToolCalls  []ToolCall
	ToolCallID string
}

// MarshalJSON writes the message as the chat API takes it: the keys are the API's, and a content,
// tool calls and tool call id that are empty are left out.
func (m Message) MarshalJSON() ([]byte, error) {
	fields := map[string]any{"role": m.Role}
	if m.Content != nil {
		fields["content"] = m.Content
	}
	if len(m.ToolCalls) > 0 {
		fields["tool_calls"] = m.ToolCalls
	}
	if m.ToolCallID != "" {
		fields["tool_call_id"] = m.ToolCallID
	}
	return json.Marshal(fields)
}

// UnmarshalJSON reads a message in the form the chat API writes it.
func (m *Message) UnmarshalJSON(data []byte) error {
	raw, err := readFields(data)
	if err != nil {
		return err
	}
	read := Message{Role: "", Content: nil, ToolCalls: nil, ToolCallID: ""}
	err = errors.Join(
		readField(raw, "role", &read.Role),
		readField(raw, "content", &read.Content),
		readField(raw, "tool_calls", &read.ToolCalls),
		readField(raw, "tool_call_id", &read.ToolCallID),
	)
	if err != nil {
		return err
	}
	*m = read
	return nil
}

func readFields(data []byte) (map[string]json.RawMessage, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func readField(raw map[string]json.RawMessage, key string, target any) error {
	value, ok := raw[key]
	if !ok {
		return nil
	}
	return json.Unmarshal(value, target)
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
