package service

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

// serverName is what the run's tools are served to the agent under.
const serverName = "numen"

// toolEndpoint is where the run's tools are served, the token presented there, and the tools the
// agent may call.
type toolEndpoint struct {
	url   string
	token string
	tools []string
}

// listServers is the MCP servers the session is opened with: the run's tools over HTTP, or none
// where the run has no tools.
func (s *acpSession) listServers() ([]any, error) {
	if s.tools.url == "" {
		return []any{}, nil
	}
	if !s.hasHTTPServers {
		return nil, fmt.Errorf("%s takes no MCP server over HTTP, so it cannot reach the tools", s.program)
	}
	return []any{map[string]any{
		"type":    "http",
		"name":    serverName,
		"url":     s.tools.url,
		"headers": []map[string]string{{"name": "Authorization", "value": "Bearer " + s.tools.token}},
	}}, nil
}

// answerPermission selects the option that allows the call once where the call is one of the run's
// tools, and refuses anything else. The tool is told by the programmatic name ACP gives a tool call,
// never by its title, which an agent writes for a person.
func (s *acpSession) answerPermission(params json.RawMessage) map[string]any {
	denial := map[string]any{"outcome": map[string]any{"outcome": "cancelled"}}
	var request struct {
		SessionID string `json:"sessionId"`
		ToolCall  struct {
			ToolCallID string `json:"toolCallId"`
			Name       string `json:"name"`
		} `json:"toolCall"`
		Options []struct {
			OptionID string `json:"optionId"`
			Kind     string `json:"kind"`
		} `json:"options"`
	}
	if json.Unmarshal(params, &request) != nil || request.SessionID != s.id || !s.isRunTool(request.ToolCall.ToolCallID, request.ToolCall.Name) {
		return denial
	}
	for _, option := range request.Options {
		if option.Kind == "allow_once" {
			return map[string]any{"outcome": map[string]any{"outcome": "selected", "optionId": option.OptionID}}
		}
	}
	return denial
}

// isRunTool says whether a tool call is a call of one of the run's tools, by the name the request
// gives it, else by the name the agent reported for the call before. A call of no name is none.
func (s *acpSession) isRunTool(callID, name string) bool {
	if name == "" {
		if call := s.calls[callID]; call != nil {
			name = call.name
		}
	}
	name = readToolName(name)
	return name != "" && slices.Contains(s.tools.tools, name)
}

// readToolEndpoint is where the task's tools are served. A task that names no tool, or no endpoint,
// has none.
func readToolEndpoint(task *agentrun.AskAgentRequest) toolEndpoint {
	if !agentrun.HasTools(task) || task.GetToolsUrl() == "" {
		return toolEndpoint{}
	}
	return toolEndpoint{url: task.GetToolsUrl(), token: task.GetToolsToken(), tools: task.GetTools()}
}
