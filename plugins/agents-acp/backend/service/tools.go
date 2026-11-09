package service

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
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

func (s *acpSession) listServers() ([]any, error) {
	if s.tools.url == "" {
		return []any{}, nil
	}
	if !s.hasHTTPServers {
		return nil, fmt.Errorf("%s takes no MCP server over HTTP, so it cannot reach the tools", s.program)
	}
	return []any{map[string]any{
		"type":    "http",
		keyName:   serverName,
		"url":     s.tools.url,
		"headers": []map[string]string{{keyName: "Authorization", keyValue: "Bearer " + s.tools.token}},
	}}, nil
}

func (s *acpSession) answerPermission(params json.RawMessage) map[string]any {
	denial := map[string]any{keyOutcome: map[string]any{keyOutcome: "canceled"}}
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
			return map[string]any{keyOutcome: map[string]any{keyOutcome: "selected", "optionId": option.OptionID}}
		}
	}
	return denial
}

func (s *acpSession) isRunTool(callID, name string) bool {
	if name == "" {
		if call := s.calls[callID]; call != nil {
			name = call.name
		}
	}
	name = readToolName(name)
	return name != "" && slices.Contains(s.tools.tools, name)
}

func readToolEndpoint(task *agentrun.Request) toolEndpoint {
	if !agentrun.HasTools(task) || task.ToolsURL == "" {
		return toolEndpoint{url: "", token: "", tools: nil}
	}
	return toolEndpoint{url: task.ToolsURL, token: task.ToolsToken, tools: task.Tools}
}
