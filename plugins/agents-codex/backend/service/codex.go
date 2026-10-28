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
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/clock"
)

// AgentModel represents an AI model available from an external agent CLI.
type AgentModel struct {
	Name      string `json:"name"`
	Title     string `json:"title"`
	IsDefault bool   `json:"isDefault,omitempty"`
	// Reasoning are the depths the model can be asked to work at, from the least to the most.
	Reasoning        []reasoning.Level `json:"-"`
	DefaultReasoning string            `json:"-"`
}

type codexCache struct {
	Models []struct {
		Slug        string `json:"slug"`
		DisplayName string `json:"display_name"`
		Visibility  string `json:"visibility"`
		Default     string `json:"default_reasoning_level"`
		Supported   []struct {
			Effort string `json:"effort"`
		} `json:"supported_reasoning_levels"`
	} `json:"models"`
}

// readCodexCache reads visible models from ~/.codex/models_cache.json.
func readCodexCache() []AgentModel {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	data, err := agentrun.ReadFile(filepath.Join(home, ".codex", "models_cache.json"))
	if err != nil {
		return nil
	}
	var cache codexCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}
	var models []AgentModel
	for _, m := range cache.Models {
		if m.Visibility == "hide" || m.Slug == "" {
			continue
		}
		levels := make([]string, 0, len(m.Supported))
		for _, one := range m.Supported {
			levels = append(levels, one.Effort)
		}
		models = append(models, newAgentModel(m.Slug, m.DisplayName, false, levels, m.Default))
	}
	return models
}

// codexListLimit is how long Codex has to list its models.
const codexListLimit = 15 * time.Second

// codexFailureBackoff is how long a command line that failed to list is left alone.
const codexFailureBackoff = 30 * time.Second

// codexModelCache caches what Codex listed, for as long as the command line asked stays the same. A
// failure caches nothing, and the command line that failed is not asked again for a while.
type codexModelCache struct {
	mu       sync.Mutex
	key      string
	models   []AgentModel
	isCached bool
	// failedKey and failedAt are the command line Codex last failed to list under, and when.
	failedKey string
	failedAt  time.Time
	inFlight  map[string]bool
	isAsync   bool
	// now is the clock the backoff uses. Empty is the system clock.
	now func() time.Time
}

// getNow is the time by the clock of the cache.
func (c *codexModelCache) getNow() time.Time {
	if c.now == nil {
		return clock.Now()
	}
	return c.now()
}

// get are the models Codex lists: the cached ones, or those it lists now.
func (c *codexModelCache) get(ctx context.Context, command []string) []AgentModel {
	key := strings.Join(command, " ")
	c.mu.Lock()
	if c.isCached && c.key == key {
		models := c.models
		c.mu.Unlock()
		return models
	}
	if c.failedKey == key && c.getNow().Sub(c.failedAt) < codexFailureBackoff {
		c.mu.Unlock()
		return nil
	}
	if c.isAsync {
		if !c.inFlight[key] {
			if c.inFlight == nil {
				c.inFlight = make(map[string]bool)
			}
			c.inFlight[key] = true
			cmdCopy := slices.Clone(command)
			go c.fetchInBackground(key, cmdCopy)
		}
		c.mu.Unlock()
		return readCodexCache()
	}
	c.mu.Unlock()
	return c.fetchAndStore(ctx, key, command)
}

func (c *codexModelCache) fetchInBackground(key string, command []string) {
	defer func() {
		c.mu.Lock()
		delete(c.inFlight, key)
		c.mu.Unlock()
	}()
	c.fetchAndStore(context.Background(), key, command)
}

func (c *codexModelCache) fetchAndStore(ctx context.Context, key string, command []string) []AgentModel {
	asking, stop := context.WithTimeout(ctx, codexListLimit)
	defer stop()
	models, err := GetCodexModels(asking, command)
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		if ctx.Err() == nil {
			c.failedKey, c.failedAt = key, c.getNow()
		}
		return nil
	}
	c.key, c.models, c.isCached = key, models, true
	c.failedKey = ""
	return models
}

// GetCodexModels returns models reported by codex app-server or its cache.
func GetCodexModels(ctx context.Context, command []string) ([]AgentModel, error) {
	models, err := askCodexModels(ctx, command)
	if len(models) > 0 {
		return models, nil
	}
	if cached := readCodexCache(); len(cached) > 0 {
		return cached, nil
	}
	return nil, err
}

func askCodexModels(ctx context.Context, command []string) ([]AgentModel, error) {
	name, args := FindCodex(), []string(nil)
	if len(command) > 0 {
		name, args = command[0], command[1:]
	}
	running, stop := context.WithCancel(ctx)
	defer stop()
	cmd := exec.CommandContext(running, name, slices.Concat(args, []string{"app-server"})...)
	supervise(cmd)
	in, err := cmd.StdinPipe()
	// gomutants:disable-next-line BRANCH_IF reason="cmd.StdinPipe cannot fail on an unstarted exec.Cmd"
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
		_ = in.Close()
		_ = out.Close()
		stop()
		_ = cmd.Wait()
	}()
	return listCodexModels(in, out)
}

func listCodexModels(in io.Writer, out io.Reader) ([]AgentModel, error) {
	lines := bufio.NewScanner(out)
	lines.Buffer(nil, 4<<20)
	hello := map[string]any{"id": 1, "method": "initialize", "params": map[string]any{
		"clientInfo": map[string]any{"name": "numen", "title": "Numen", "version": "1.0.0"},
	}}
	if err := writeCodexLine(in, hello); err != nil {
		return nil, err
	}
	if _, err := readCodexAnswer(lines, 1); err != nil {
		return nil, err
	}
	if err := writeCodexLine(in, map[string]any{"method": "initialized", "params": map[string]any{}}); err != nil {
		return nil, err
	}
	if err := writeCodexLine(in, map[string]any{"id": 2, "method": "model/list", "params": map[string]any{}}); err != nil {
		return nil, err
	}
	result, err := readCodexAnswer(lines, 2)
	if err != nil {
		return nil, err
	}
	var listing struct {
		Data []struct {
			ID            string `json:"id"`
			Model         string `json:"model"`
			DisplayName   string `json:"displayName"`
			Hidden        bool   `json:"hidden"`
			IsDefault     bool   `json:"isDefault"`
			DefaultEffort string `json:"defaultReasoningEffort"`
			Efforts       []struct {
				Effort string `json:"reasoningEffort"`
			} `json:"supportedReasoningEfforts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(result, &listing); err != nil {
		return nil, err
	}
	var models []AgentModel
	for _, one := range listing.Data {
		name := one.Model
		if name == "" {
			name = one.ID
		}
		if one.Hidden || name == "" {
			continue
		}
		levels := make([]string, 0, len(one.Efforts))
		for _, effort := range one.Efforts {
			levels = append(levels, effort.Effort)
		}
		models = append(models, newAgentModel(name, one.DisplayName, one.IsDefault, levels, one.DefaultEffort))
	}
	return models, nil
}

func writeCodexLine(in io.Writer, message map[string]any) error {
	raw, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = in.Write(append(raw, '\n'))
	return err
}

func readCodexAnswer(lines *bufio.Scanner, id int) (json.RawMessage, error) {
	for lines.Scan() {
		var reply struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(lines.Bytes(), &reply); err != nil || reply.ID != id {
			continue
		}
		if reply.Error != nil {
			return nil, errors.New(reply.Error.Message)
		}
		return reply.Result, nil
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("codex exited before answering message %d", id)
}

// newAgentModel is a model with the levels of reasoning it lists, the empty names left out and the
// default kept where it is one of them.
func newAgentModel(name, title string, isDefault bool, efforts []string, byDefault string) AgentModel {
	var levels []reasoning.Level
	for _, effort := range efforts {
		if level := reasoning.NewLevel(effort); level.Name != "" && !reasoning.Contains(levels, level.Name) {
			levels = append(levels, level)
		}
	}
	reasoning.Sort(levels)
	return AgentModel{
		Name: name, Title: title, IsDefault: isDefault,
		Reasoning: levels, DefaultReasoning: reasoning.Pick(levels, byDefault),
	}
}
