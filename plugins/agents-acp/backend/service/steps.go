package service

import (
	"encoding/json"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/toolcall"
)

// acpCall is one tool call of a session as the agent reported it so far.
type acpCall struct {
	// name is the programmatic name of the tool, without the prefix of an MCP server.
	name       string
	title      string
	arguments  string
	isFinished bool
}

// acpToolCall is the part of a tool_call or tool_call_update a step is made of. Every field but
// the id is optional in an update, and one left out keeps what the call had.
type acpToolCall struct {
	SessionUpdate string          `json:"sessionUpdate"`
	ToolCallID    string          `json:"toolCallId"`
	Name          *string         `json:"name"`
	Title         *string         `json:"title"`
	Status        *string         `json:"status"`
	RawInput      json.RawMessage `json:"rawInput"`
}

// readToolStep keeps what a tool_call or tool_call_update of this session says of the call, and
// returns the step it is: the call, as far as it is known, or its end once it completed or failed.
// Any other update is no step. A call is reported finished once.
func (s *acpSession) readToolStep(params json.RawMessage) *agentrun.AskAgentResponse {
	var update struct {
		SessionID string      `json:"sessionId"`
		Update    acpToolCall `json:"update"`
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
		call = &acpCall{}
		s.calls[reported.ToolCallID] = call
	}
	call.update(reported)
	if call.isFinished {
		return nil
	}
	if reported.Status != nil && (*reported.Status == "completed" || *reported.Status == "failed") {
		call.isFinished = true
		return &agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponse_ToolFinished{ToolFinished: &agentrun.ToolFinished{}}}
	}
	return &agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponse_ToolCall{
		ToolCall: toolcall.Describe(s.words, call.getShownName(), reported.ToolCallID, call.arguments),
	}}
}

// update takes in what the agent reported of the call.
func (c *acpCall) update(reported acpToolCall) {
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

// getShownName is the tool by its name, else by the title the agent gave the call.
func (c *acpCall) getShownName() string {
	if c.name != "" {
		return c.name
	}
	return c.title
}

// readToolName is the name of a tool without the prefix an agent gives a tool of an MCP server,
// mcp__<server>__<tool>.
func readToolName(name string) string {
	if tool, isServed := strings.CutPrefix(name, "mcp__"+serverName+"__"); isServed {
		return tool
	}
	return name
}
