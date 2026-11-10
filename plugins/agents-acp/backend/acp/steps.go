package acp

import (
	"encoding/json"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
)

// trackedCall is one tool call of a session as the agent reported it so far.
type trackedCall struct {
	name       string
	title      string
	arguments  string
	isFinished bool
}

// toolUpdate is the part of a tool_call or tool_call_update a step is made of. Every field but
// the id is optional in an update, and one left out keeps what the call had.
type toolUpdate struct {
	SessionUpdate string          `json:"sessionUpdate"`
	ToolCallID    string          `json:"toolCallId"`
	Name          *string         `json:"name"`
	Title         *string         `json:"title"`
	Status        *string         `json:"status"`
	RawInput      json.RawMessage `json:"rawInput"`
}

func (s *Session) readToolStep(params json.RawMessage) *agentrun.Response {
	var update struct {
		SessionID string     `json:"sessionId"`
		Update    toolUpdate `json:"update"`
	}
	if json.Unmarshal(params, &update) != nil || update.SessionID != s.id || update.Update.ToolCallID == "" {
		return nil
	}
	reported := update.Update
	if reported.SessionUpdate != "tool_call" && reported.SessionUpdate != "tool_call_update" {
		return nil
	}
	call := s.calls[reported.ToolCallID]
	if call == nil {
		call = &trackedCall{name: "", title: "", arguments: "", isFinished: false}
		s.calls[reported.ToolCallID] = call
	}
	call.update(reported)
	if call.isFinished {
		return nil
	}
	if reported.Status != nil && (*reported.Status == "completed" || *reported.Status == "failed") {
		call.isFinished = true
		return &agentrun.Response{Step: &agentrun.ToolFinishedStep{}}
	}
	return &agentrun.Response{Step: &agentrun.ToolCallStep{
		ToolCall: toolcall.Describe(s.words, call.getShownName(), reported.ToolCallID, call.arguments),
	}}
}

func (c *trackedCall) update(reported toolUpdate) {
	if reported.Name != nil && *reported.Name != "" {
		c.name = readToolName(*reported.Name)
	}
	if reported.Title != nil {
		c.title = *reported.Title
	}
	if len(reported.RawInput) > 0 && string(reported.RawInput) != "null" {
		c.arguments = string(reported.RawInput)
	}
}

func (c *trackedCall) getShownName() string {
	if c.name != "" {
		return c.name
	}
	return c.title
}

func readToolName(name string) string {
	if tool, isServed := strings.CutPrefix(name, "mcp__"+serverName+"__"); isServed {
		return tool
	}
	return name
}
