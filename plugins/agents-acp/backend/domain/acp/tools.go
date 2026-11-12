package acp

import (
	"encoding/json"
	"fmt"
	"slices"
)

// serverName is what the run's tools are served to the agent under.
const serverName = "numen"

// Tools is where the run's tools are served, the token presented there, and the names of the tools
// the agent may call.
type Tools struct {
	URL   string
	Token string
	Names []string
}

func (s *Session) listServers() ([]any, error) {
	if s.tools.URL == "" {
		return []any{}, nil
	}
	if !s.hasHTTPServers {
		return nil, fmt.Errorf("%s takes no MCP server over HTTP, so it cannot reach the tools", s.program)
	}
	return []any{map[string]any{
		"type":    "http",
		keyName:   serverName,
		"url":     s.tools.URL,
		"headers": []map[string]string{{keyName: "Authorization", keyValue: "Bearer " + s.tools.Token}},
	}}, nil
}

func (s *Session) answerPermission(params json.RawMessage) map[string]any {
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

func (s *Session) isRunTool(callID, name string) bool {
	if name == "" {
		if call := s.calls[callID]; call != nil {
			name = call.name
		}
	}
	name = readToolName(name)
	return name != "" && slices.Contains(s.tools.Names, name)
}
