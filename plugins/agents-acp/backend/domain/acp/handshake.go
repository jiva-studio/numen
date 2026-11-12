package acp

import (
	"context"
	"encoding/json"
	"fmt"
)

func (s *Session) initialize(ctx context.Context) error {
	result, err := s.call(ctx, "initialize", map[string]any{
		"protocolVersion": protocolVersion,
		"clientCapabilities": map[string]any{
			"fs":       map[string]any{"readTextFile": false, "writeTextFile": false},
			"terminal": false,
		},
		"clientInfo": map[string]any{keyName: "numen", "title": "Numen", "version": "1.0.0"},
	}, nil)
	if err != nil {
		return err
	}
	var reply struct {
		AgentCapabilities struct {
			SessionCapabilities struct {
				Close json.RawMessage `json:"close"`
			} `json:"sessionCapabilities"`
			MCPCapabilities struct {
				HTTP bool `json:"http"`
			} `json:"mcpCapabilities"`
		} `json:"agentCapabilities"`
	}
	if err := json.Unmarshal(result, &reply); err != nil {
		return fmt.Errorf("read the initialize reply of %s: %w", s.program, err)
	}
	closing := string(reply.AgentCapabilities.SessionCapabilities.Close)
	s.canClose = closing != "" && closing != "null" && closing != "false"
	s.hasHTTPServers = reply.AgentCapabilities.MCPCapabilities.HTTP
	return nil
}

func (s *Session) create(ctx context.Context, start Start) error {
	params := map[string]any{"cwd": s.proc.Dir(), "mcpServers": listServers(start.Servers)}
	if start.Meta != nil {
		params["_meta"] = start.Meta
	}
	result, err := s.call(ctx, "session/new", params, nil)
	if err != nil {
		return err
	}
	var reply struct {
		SessionID     string         `json:"sessionId"`
		ConfigOptions []configOption `json:"configOptions"`
		Models        *struct {
			AvailableModels []struct {
				ModelID string `json:"modelId"`
				Name    string `json:"name"`
			} `json:"availableModels"`
			CurrentModelID string `json:"currentModelId"`
		} `json:"models"`
	}
	if err := json.Unmarshal(result, &reply); err != nil {
		return fmt.Errorf("read the session of %s: %w", s.program, err)
	}
	if reply.SessionID == "" {
		return fmt.Errorf("%s opened a session with no id", s.program)
	}
	s.id = reply.SessionID
	s.readConfigOptions(reply.ConfigOptions)
	if s.modelOption == "" && reply.Models != nil {
		s.current = reply.Models.CurrentModelID
		for _, one := range reply.Models.AvailableModels {
			s.models = append(s.models, Model{ID: one.ModelID, Name: one.Name})
		}
	}
	return nil
}

func listServers(servers []Server) []any {
	listed := make([]any, 0, len(servers))
	for _, one := range servers {
		listed = append(listed, map[string]any{
			"type":    "http",
			keyName:   one.Name,
			"url":     one.URL,
			"headers": []map[string]string{{keyName: "Authorization", keyValue: "Bearer " + one.Token}},
		})
	}
	return listed
}

func (s *Session) readConfigOptions(options []configOption) {
	for _, option := range options {
		if option.Category == "thought_level" && option.Type == "select" {
			s.thoughtOption, s.thoughtCurrent, s.thoughtLevels = option.ID, option.CurrentValue, option.getLevels()
		}
		if option.Category == "model" && option.Type == "select" {
			s.modelOption, s.current, s.models = option.ID, option.CurrentValue, option.getModels()
		}
	}
}
