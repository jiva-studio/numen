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

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/process"
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

func (m claudeModel) getLevels() []reasoning.Level {
	if !m.SupportsEffort {
		return nil
	}
	levels := reasoning.NewLevels(m.EffortLevels...)
	reasoning.Sort(levels)
	return levels
}

func (m claudeModel) toProto() *reasoning.Model {
	return reasoning.NewModel(m.Value, m.DisplayName, m.getLevels(), "")
}

func getFallbackModels() []claudeModel {
	return []claudeModel{
		{Value: "sonnet", DisplayName: "Sonnet", SupportsEffort: false, EffortLevels: nil},
		{Value: "opus", DisplayName: "Opus", SupportsEffort: false, EffortLevels: nil},
		{Value: "haiku", DisplayName: "Haiku", SupportsEffort: false, EffortLevels: nil},
	}
}

func parseModels(lines io.Reader, requestID string) ([]claudeModel, error) {
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

func isStopped(err error) bool {
	var exit *exec.ExitError
	return err == nil || errors.As(err, &exit) || errors.Is(err, context.Canceled) ||
		errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH)
}

func fetchModels(ctx context.Context, command, env []string) (models []claudeModel, err error) {
	ctx, cancel := context.WithTimeout(ctx, modelsTimeout)
	defer cancel()
	args := slices.Concat(command[1:], []string{
		"-p", "--input-format", streamJSON, flagOutputFormat, streamJSON, flagVerbose,
		"--setting-sources", "",
	})
	cmd := exec.CommandContext(ctx, command[0], args...)
	cmd.Env = buildEnvironment(env)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="process.Supervise sets OS process-group attributes for child cleanup."
	process.Supervise(cmd)
	in, err := cmd.StdinPipe()
	// gomutants:disable-next-line BRANCH_IF reason="cmd.StdinPipe on a fresh command before Start fails only on OS descriptor exhaustion."
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	// gomutants:disable-next-line BRANCH_IF reason="cmd.StdoutPipe on a fresh command before Start fails only on OS descriptor exhaustion."
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Closing the child stdin pipe in defer is best-effort cleanup after reading models."
		err = errors.Join(err, in.Close())
		// gomutants:disable-next-line BRANCH_IF reason="WaitForExit after Start returns only OS wait errors that are covered by isStopped."
		if waitErr := process.WaitForExit(cmd, cancel); !isStopped(waitErr) {
			err = errors.Join(err, waitErr)
		}
	}()
	const requestID = "numen-models"
	request := `{"type":"control_request","request_id":"` + requestID + `","request":{"subtype":"initialize"}}` + "\n"
	// gomutants:disable-next-line BRANCH_IF reason="Writing the small initialize payload to a newly started pipe fails only if the child closes stdin immediately, which parseModels also detects."
	if _, err := io.WriteString(in, request); err != nil {
		return nil, err
	}
	return parseModels(out, requestID)
}

func newModelCache(isAsync bool, extra ...cache.Option) *cache.Cache[[]claudeModel] {
	return cache.New[[]claudeModel](
		"models",
		slices.Concat([]cache.Option{
			cache.WithTTL(modelsLifetime),
			cache.WithTimeout(modelsTimeout),
			cache.WithAsync(isAsync),
		}, extra)...,
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

func (s *agentServer) getModels(ctx context.Context, command, env []string) ([]claudeModel, error) {
	key := strings.Join(command, "\x00")
	modelCache := s.getModelCache()
	cmdCopy := slices.Clone(command)
	envCopy := slices.Clone(env)
	fetch := s.fetch
	if fetch == nil {
		fetch = fetchModels
	}
	models, found, err := modelCache.Get(ctx, key, func(ctx context.Context) ([]claudeModel, error) {
		return fetch(ctx, cmdCopy, envCopy)
	})
	if err != nil {
		return nil, err
	}
	if !found && modelCache.HasFailed(key) {
		return getFallbackModels(), nil
	}
	return models, nil
}

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
