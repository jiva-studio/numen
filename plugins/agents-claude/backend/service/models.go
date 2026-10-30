package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
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
func (m claudeModel) toProto() *agentrun.AgentModel {
	return reasoning.NewModel(m.Value, m.DisplayName, m.getLevels(), "")
}

// fallbackModels are the aliases the program accepts for --model, offered with no depth of reasoning
// when the program does not answer the request for its models.
var fallbackModels = []claudeModel{
	{Value: "sonnet", DisplayName: "Sonnet", SupportsEffort: false, EffortLevels: nil},
	{Value: "opus", DisplayName: "Opus", SupportsEffort: false, EffortLevels: nil},
	{Value: "haiku", DisplayName: "Haiku", SupportsEffort: false, EffortLevels: nil},
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

// isStopped reports whether err is what Wait returns for a program that was stopped: no error, a
// non-zero exit status, the cancellation of the program's context, or a process that is already gone.
func isStopped(err error) bool {
	var exit *exec.ExitError
	return err == nil || errors.As(err, &exit) || errors.Is(err, context.Canceled) ||
		errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH)
}

// fetchModels queries the program, started as command with env, for the models it offers.
func fetchModels(ctx context.Context, command, env []string) (models []claudeModel, err error) {
	ctx, cancel := context.WithTimeout(ctx, modelsTimeout)
	defer cancel()
	args := slices.Concat(command[1:], []string{
		"-p", "--input-format", streamJSON, flagOutputFormat, streamJSON, flagVerbose,
		"--setting-sources", "",
	})
	cmd := exec.CommandContext(ctx, command[0], args...)
	cmd.Env = buildEnvironment(env)
	supervise(cmd)
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
		if waitErr := cmd.Wait(); !isStopped(waitErr) {
			err = errors.Join(err, waitErr)
		}
	}()
	const requestID = "numen-models"
	request := `{"type":"control_request","request_id":"` + requestID + `","request":{"subtype":"initialize"}}` + "\n"
	if _, err := io.WriteString(in, request); err != nil {
		return nil, err
	}
	return parseModels(out, requestID)
}

func newModelCache(isAsync bool) *cache.Cache[[]claudeModel] {
	return cache.New[[]claudeModel](
		"models",
		cache.WithPluginID("agents-claude"),
		cache.WithTTL(modelsLifetime),
		cache.WithTimeout(modelsTimeout),
		cache.WithAsync(isAsync),
	)
}

func (s *agentServer) getModelCache() *cache.Cache[[]claudeModel] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.models == nil {
		s.models = newModelCache(false)
	}
	return s.models
}

// getModels returns the current list of models for the command, listing them where none is current.
func (s *agentServer) getModels(ctx context.Context, command, env []string) ([]claudeModel, error) {
	key := strings.Join(command, "\x00")
	modelCache := s.getModelCache()
	cmdCopy := slices.Clone(command)
	envCopy := slices.Clone(env)
	models, found, err := modelCache.Get(ctx, key, func(ctx context.Context) ([]claudeModel, error) {
		return fetchModels(ctx, cmdCopy, envCopy)
	})
	if err != nil {
		return nil, err
	}
	if !found && modelCache.HasFailed(key) {
		return fallbackModels, nil
	}
	return models, nil
}

// findModelLevels returns the levels of the model among the current lists, and whether the model is listed.
func (s *agentServer) findModelLevels(model string) ([]reasoning.Level, bool) {
	for _, list := range s.getModelCache().Values() {
		for _, items := range list {
			if items.Value == model {
				return items.getLevels(), true
			}
		}
	}
	return nil, false
}
