package acp

import (
	"encoding/json"
	"fmt"
)

func (s *Session) initialize() error {
	result, err := s.call("initialize", map[string]any{
		"protocolVersion": protocolVersion,
		"clientCapabilities": map[string]any{
			"fs":       map[string]any{"readTextFile": false, "writeTextFile": false},
			"terminal": false,
		},
		"clientInfo": map[string]any{keyName: "numen", "title": "Numen", "version": "1.0.0"},
	})
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

func (s *Session) create() error {
	servers, err := s.listServers()
	if err != nil {
		return err
	}
	result, err := s.call("session/new", map[string]any{"cwd": s.proc.Dir(), "mcpServers": servers})
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
