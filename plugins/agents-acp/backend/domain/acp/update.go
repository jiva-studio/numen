package acp

import (
	"encoding/json"
	"strings"
)

// StopReason is why the program ended a turn, as the protocol names it.
type StopReason string

// The stop reasons of the protocol.
const (
	StopEndTurn         StopReason = "end_turn"
	StopMaxTokens       StopReason = "max_tokens"
	StopMaxTurnRequests StopReason = "max_turn_requests"
	StopRefusal         StopReason = "refusal"
)

// IsCanceled reports whether the program ended the turn because it was canceled.
func (r StopReason) IsCanceled() bool {
	return strings.HasPrefix(string(r), "cancel")
}

// ServerName is the name the MCP server of the tools of a session is offered under.
const ServerName = "numen"

// Update is one session update of a turn: exactly one of its fields is set.
type Update struct {
	Chunk *Chunk
	Plan  *Plan
	Tool  *ToolUpdate
}

// Chunk is a piece of the answer of the agent. MessageID is empty where the agent names no message.
type Chunk struct {
	MessageID string
	Text      string
}

// Plan is the plan the agent reports, whole each time.
type Plan struct {
	Entries []PlanEntry
}

// PlanEntry is one step of a plan, with the priority and the status the protocol names.
type PlanEntry struct {
	Content  string
	Priority string
	Status   string
}

// ToolUpdate is a tool_call (IsStart) or a tool_call_update. A field left nil or empty keeps what the
// call had.
type ToolUpdate struct {
	ID      string
	IsStart bool
	// Name is the name of the tool the update gives, without the prefix the agent adds to a tool of the
	// MCP server of the session; empty where the update gives none.
	Name   string
	Title  *string
	Kind   *string
	Status *string
	// Arguments are the arguments of the tool as one JSON value.
	Arguments json.RawMessage
	Output    json.RawMessage
}

// rawToolUpdate is the part of a tool_call, a tool_call_update or the tool call of a permission request
// that the client reads.
type rawToolUpdate struct {
	SessionUpdate string          `json:"sessionUpdate"`
	ToolCallID    string          `json:"toolCallId"`
	Name          *string         `json:"name"`
	Title         *string         `json:"title"`
	Kind          *string         `json:"kind"`
	Status        *string         `json:"status"`
	RawInput      json.RawMessage `json:"rawInput"`
	RawOutput     json.RawMessage `json:"rawOutput"`
	Meta          struct {
		ClaudeCode struct {
			ToolName string `json:"toolName"`
		} `json:"claudeCode"`
	} `json:"_meta"`
}

// readServedTool returns the tool of the session's MCP server the call names and the arguments of the
// call. Claude Code names it in _meta, Codex wraps the arguments with the server and the tool, and other
// agents name it in name.
func (r rawToolUpdate) readServedTool() (string, json.RawMessage) {
	arguments := r.RawInput
	if !isValue(arguments) {
		arguments = nil
	}
	if name, isServed := cutServerPrefix(r.Meta.ClaudeCode.ToolName); isServed {
		return name, arguments
	}
	var wrapped struct {
		Server    string          `json:"server"`
		Tool      string          `json:"tool"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if arguments != nil && json.Unmarshal(arguments, &wrapped) == nil && wrapped.Server == ServerName && wrapped.Tool != "" {
		return wrapped.Tool, wrapped.Arguments
	}
	if r.Name != nil && *r.Name != "" {
		name, _ := cutServerPrefix(*r.Name)
		return name, arguments
	}
	return "", arguments
}

func cutServerPrefix(name string) (string, bool) {
	return strings.CutPrefix(name, "mcp__"+ServerName+"__")
}

func isValue(raw json.RawMessage) bool {
	return len(raw) > 0 && string(raw) != "null"
}

func (s *Session) readUpdate(params json.RawMessage, handler Handler) error {
	if handler == nil {
		return nil
	}
	var envelope struct {
		SessionID string          `json:"sessionId"`
		Update    json.RawMessage `json:"update"`
	}
	if !isDecoded(params, &envelope) || envelope.SessionID != s.id {
		return nil
	}
	var head struct {
		SessionUpdate string `json:"sessionUpdate"`
	}
	if !isDecoded(envelope.Update, &head) {
		return nil
	}
	var update *Update
	switch head.SessionUpdate {
	case "agent_message_chunk":
		update = readChunk(envelope.Update)
	case "plan":
		update = readPlan(envelope.Update)
	case "tool_call", "tool_call_update":
		update = s.readToolUpdate(envelope.Update)
	}
	if update == nil {
		return nil
	}
	return handler.HandleUpdate(*update)
}

// isDecoded decodes raw into value and reports whether it could; an update the client cannot read is
// left out.
func isDecoded(raw json.RawMessage, value any) bool {
	return json.Unmarshal(raw, value) == nil
}

func readChunk(raw json.RawMessage) *Update {
	var chunk struct {
		MessageID string `json:"messageId"`
		Content   struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if json.Unmarshal(raw, &chunk) != nil || chunk.Content.Type != keyText || chunk.Content.Text == "" {
		return nil
	}
	return &Update{Chunk: &Chunk{MessageID: chunk.MessageID, Text: chunk.Content.Text}, Plan: nil, Tool: nil}
}

func readPlan(raw json.RawMessage) *Update {
	var plan struct {
		Entries []struct {
			Content  string `json:"content"`
			Priority string `json:"priority"`
			Status   string `json:"status"`
		} `json:"entries"`
	}
	if json.Unmarshal(raw, &plan) != nil {
		return nil
	}
	entries := make([]PlanEntry, 0, len(plan.Entries))
	for _, one := range plan.Entries {
		entries = append(entries, PlanEntry{Content: one.Content, Priority: one.Priority, Status: one.Status})
	}
	return &Update{Chunk: nil, Plan: &Plan{Entries: entries}, Tool: nil}
}

func (s *Session) readToolUpdate(raw json.RawMessage) *Update {
	var reported rawToolUpdate
	if json.Unmarshal(raw, &reported) != nil || reported.ToolCallID == "" {
		return nil
	}
	name, arguments := reported.readServedTool()
	if name != "" {
		s.names[reported.ToolCallID] = name
	}
	output := reported.RawOutput
	if !isValue(output) {
		output = nil
	}
	return &Update{Chunk: nil, Plan: nil, Tool: &ToolUpdate{
		ID: reported.ToolCallID, IsStart: reported.SessionUpdate == "tool_call", Name: name,
		Title: reported.Title, Kind: reported.Kind, Status: reported.Status,
		Arguments: arguments, Output: output,
	}}
}

// answerPermission grants a permission once where the handler allows the tool the request names, or,
// where it names none, the tool the agent reported for the call. It refuses every other request with the
// option that rejects once, which lets the turn go on, and returns nil for a request that offers no
// option to reject.
func (s *Session) answerPermission(params json.RawMessage, handler Handler) map[string]any {
	var request struct {
		SessionID string        `json:"sessionId"`
		ToolCall  rawToolUpdate `json:"toolCall"`
		Options   []struct {
			OptionID string `json:"optionId"`
			Kind     string `json:"kind"`
		} `json:"options"`
	}
	if !isDecoded(params, &request) {
		return nil
	}
	kinds := make(map[string]string, len(request.Options))
	for _, option := range request.Options {
		if _, isListed := kinds[option.Kind]; !isListed {
			kinds[option.Kind] = option.OptionID
		}
	}
	name, _ := request.ToolCall.readServedTool()
	if name == "" {
		name = s.names[request.ToolCall.ToolCallID]
	}
	if handler != nil && request.SessionID == s.id && name != "" && handler.IsAllowed(name) {
		if option, isOffered := kinds["allow_once"]; isOffered {
			return selectOption(option)
		}
	}
	for _, kind := range []string{"reject_once", "reject_always"} {
		if option, isOffered := kinds[kind]; isOffered {
			return selectOption(option)
		}
	}
	return nil
}

// selectOption answers a permission request with the option.
func selectOption(option string) map[string]any {
	return map[string]any{keyOutcome: map[string]any{keyOutcome: "selected", "optionId": option}}
}
