package service

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
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
// tools, and refuses anything else.
func (s *acpSession) answerPermission(params json.RawMessage) map[string]any {
	denial := map[string]any{"outcome": map[string]any{"outcome": "cancelled"}}
	var request struct {
		SessionID string `json:"sessionId"`
		ToolCall  struct {
			Title string `json:"title"`
		} `json:"toolCall"`
		Options []struct {
			OptionID string `json:"optionId"`
			Kind     string `json:"kind"`
		} `json:"options"`
	}
	if json.Unmarshal(params, &request) != nil || request.SessionID != s.id || !s.isRunTool(request.ToolCall.Title) {
		return denial
	}
	for _, option := range request.Options {
		if option.Kind == "allow_once" {
			return map[string]any{"outcome": map[string]any{"outcome": "selected", "optionId": option.OptionID}}
		}
	}
	return denial
}

// isRunTool says whether a tool call's title names one of the run's tools: by its name, or by the
// name an agent gives a tool of an MCP server, mcp__<server>__<tool>.
func (s *acpSession) isRunTool(title string) bool {
	return slices.ContainsFunc(s.tools.tools, func(name string) bool {
		return title == name || title == "mcp__"+serverName+"__"+name
	})
}

// readToolEndpoint is where the task's tools are served. A task that names no tool, or no endpoint,
// has none.
func readToolEndpoint(task *numenv1.AskAgentRequest) toolEndpoint {
	if !agentrun.HasTools(task) || task.GetToolsUrl() == "" {
		return toolEndpoint{}
	}
	return toolEndpoint{url: task.GetToolsUrl(), token: task.GetToolsToken(), tools: task.GetTools()}
}
