// Package models holds the models Claude Code lists and the depths of reasoning each supports.
package models

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
)

// DefaultName is the model the program lists for the choice of its own.
const DefaultName = "default"

// Timeout is how long the program has to answer the request for its models.
const Timeout = 30 * time.Second

// Lifetime is how long a list of models stays current.
const Lifetime = 10 * time.Minute

// Model is a model as the program lists it in its answer to the initialize request of its
// control protocol, with the depths of reasoning it supports.
type Model struct {
	Value          string   `json:"value"`
	DisplayName    string   `json:"displayName"`
	SupportsEffort bool     `json:"supportsEffort"`
	EffortLevels   []string `json:"supportedEffortLevels"`
}

// GetLevels returns the depths of reasoning the model supports, shallowest first.
func (m Model) GetLevels() []reasoning.Level {
	if !m.SupportsEffort {
		return nil
	}
	levels := reasoning.NewLevels(m.EffortLevels...)
	reasoning.Sort(levels)
	return levels
}

// GetReasoningModel returns the model as the agent service lists it.
func (m Model) GetReasoningModel() *reasoning.Model {
	return reasoning.NewModel(m.Value, m.DisplayName, m.GetLevels(), "")
}

// GetFallback returns the models listed when the program does not answer.
func GetFallback() []Model {
	return []Model{
		{Value: "sonnet", DisplayName: "Sonnet", SupportsEffort: false, EffortLevels: nil},
		{Value: "opus", DisplayName: "Opus", SupportsEffort: false, EffortLevels: nil},
		{Value: "haiku", DisplayName: "Haiku", SupportsEffort: false, EffortLevels: nil},
	}
}

// Parse reads the models from the answer to the request with the given id in the output lines of
// the program.
func Parse(lines io.Reader, requestID string) ([]Model, error) {
	scanner := bufio.NewScanner(lines)
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT,ARITHMETIC_BASE reason="Scanner initial buffer allocation and maximum token size are capacity bounds."
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var line struct {
			Type     string `json:"type"`
			Response struct {
				Subtype   string `json:"subtype"`
				RequestID string `json:"request_id"`
				Error     string `json:"error"`
				Response  struct {
					Models []Model `json:"models"`
				} `json:"response"`
			} `json:"response"`
		}
		if json.Unmarshal(scanner.Bytes(), &line) != nil ||
			line.Type != "control_response" || line.Response.RequestID != requestID {
			continue
		}
		if line.Response.Subtype != "success" {
			return nil, fmt.Errorf("the program rejected the request: %s", line.Response.Error)
		}
		if len(line.Response.Response.Models) == 0 {
			return nil, errors.New("the program listed no model")
		}
		return line.Response.Response.Models, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("the program ended without listing its models")
}
