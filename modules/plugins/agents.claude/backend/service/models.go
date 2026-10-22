package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/reasoning"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// defaultModelName is the model the program lists for the choice of its own.
const defaultModelName = "default"

// modelsTimeout is how long the program has to answer the request for its models.
const modelsTimeout = 30 * time.Second

// modelsLifetime is how long a list of models stays current.
const modelsLifetime = 10 * time.Minute

// claudeModel is a model as the program lists it in its answer to the initialize request of its
// control protocol, with the depths of reasoning it supports.
type claudeModel struct {
	Value          string   `json:"value"`
	DisplayName    string   `json:"displayName"`
	SupportsEffort bool     `json:"supportsEffort"`
	EffortLevels   []string `json:"supportedEffortLevels"`
}

// getLevels returns the depths of reasoning the model supports, from the least to the most.
func (m claudeModel) getLevels() []reasoning.Level {
	if !m.SupportsEffort {
		return nil
	}
	levels := reasoning.NewLevels(m.EffortLevels...)
	reasoning.Sort(levels)
	return levels
}

// toProto returns the model in the schema's words.
func (m claudeModel) toProto() *numenv1.AgentModel {
	return reasoning.NewModel(m.Value, m.DisplayName, m.getLevels(), "")
}

// fallbackModels are the aliases the program accepts for --model, offered with no depth of reasoning
// when the program does not answer the request for its models.
var fallbackModels = []claudeModel{
	{Value: "sonnet", DisplayName: "Sonnet"},
	{Value: "opus", DisplayName: "Opus"},
	{Value: "haiku", DisplayName: "Haiku"},
}

// parseModels reads the models out of the lines the program writes: the answer to the request
// named by requestID holds them.
func parseModels(lines io.Reader, requestID string) ([]claudeModel, error) {
	scanner := bufio.NewScanner(lines)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var line struct {
			Type     string `json:"type"`
			Response struct {
				Subtype   string `json:"subtype"`
				RequestID string `json:"request_id"`
				Error     string `json:"error"`
				Response  struct {
					Models []claudeModel `json:"models"`
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

// fetchModels asks the program, started as command with env, for the models it offers.
func fetchModels(ctx context.Context, command, env []string) ([]claudeModel, error) {
	ctx, cancel := context.WithTimeout(ctx, modelsTimeout)
	defer cancel()
	args := slices.Concat(command[1:], []string{
		"-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose",
		"--setting-sources", "",
	})
	cmd := exec.CommandContext(ctx, command[0], args...)
	cmd.Env = buildEnvironment(env)
	agentrun.Supervise(cmd)
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() {
		cancel()
		_ = cmd.Wait()
	}()
	const requestID = "numen-models"
	request := `{"type":"control_request","request_id":"` + requestID + `","request":{"subtype":"initialize"}}` + "\n"
	if _, err := io.WriteString(in, request); err != nil {
		return nil, err
	}
	return parseModels(out, requestID)
}

// modelList is the models of one command and when they were read.
type modelList struct {
	models []claudeModel
	at     time.Time
}

// modelCache keeps the lists of models by the command that listed them.
type modelCache struct {
	mu    sync.Mutex
	lists map[string]modelList
}

// get returns the current list of models for the command, listing them where none is current.
func (c *modelCache) get(ctx context.Context, command, env []string) ([]claudeModel, error) {
	key := strings.Join(command, "\x00")
	c.mu.Lock()
	one, found := c.lists[key]
	c.mu.Unlock()
	if found && time.Since(one.at) < modelsLifetime {
		return one.models, nil
	}
	models, err := fetchModels(ctx, command, env)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lists == nil {
		c.lists = map[string]modelList{}
	}
	c.lists[key] = modelList{models: models, at: time.Now()}
	return models, nil
}

// find returns the levels of the model among the current lists, and whether the model is listed.
func (c *modelCache) find(model string) ([]reasoning.Level, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, one := range c.lists {
		for _, items := range one.models {
			if items.Value == model {
				return items.getLevels(), true
			}
		}
	}
	return nil, false
}
