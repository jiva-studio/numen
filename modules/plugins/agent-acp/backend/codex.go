package main

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
)

// AgentModel represents an AI model available from an external agent CLI.
type AgentModel struct {
	Name      string `json:"name"`
	Title     string `json:"title"`
	IsDefault bool   `json:"isDefault,omitempty"`
}

type codexCache struct {
	Models []struct {
		Slug        string `json:"slug"`
		DisplayName string `json:"display_name"`
		Visibility  string `json:"visibility"`
	} `json:"models"`
}

// readCodexCache reads visible models from ~/.codex/models_cache.json.
func readCodexCache() []AgentModel {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(home, ".codex", "models_cache.json"))
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
		models = append(models, AgentModel{Name: m.Slug, Title: m.DisplayName})
	}
	return models
}

// codexListLimit is how long Codex has to list its models.
const codexListLimit = 15 * time.Second

// codexFailureBackoff is how long a command line that failed to list is left alone.
const codexFailureBackoff = 30 * time.Second

// codexModelCache keeps what Codex listed, for as long as the command line asked stays the same. A
// failure keeps nothing, and the command line that failed is left alone for a while.
type codexModelCache struct {
	mu     sync.Mutex
	key    string
	models []AgentModel
	isKept bool
	// failedKey and failedAt are the command line Codex last failed to list under, and when.
	failedKey string
	failedAt  time.Time
	// now is the clock the backoff is kept by. Empty is the system clock.
	now func() time.Time
}

// getNow is the time by the clock of the cache.
func (c *codexModelCache) getNow() time.Time {
	if c.now == nil {
		return time.Now()
	}
	return c.now()
}

// get are the models Codex lists: the ones kept, or those it lists now.
func (c *codexModelCache) get(ctx context.Context, command []string) []AgentModel {
	key := strings.Join(command, " ")
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isKept && c.key == key {
		return c.models
	}
	if c.failedKey == key && c.getNow().Sub(c.failedAt) < codexFailureBackoff {
		return nil
	}
	asking, stop := context.WithTimeout(ctx, codexListLimit)
	defer stop()
	models, err := GetCodexModels(asking, command)
	if err != nil {
		if ctx.Err() == nil {
			c.failedKey, c.failedAt = key, c.getNow()
		}
		return nil
	}
	c.key, c.models, c.isKept = key, models, true
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
			ID          string `json:"id"`
			Model       string `json:"model"`
			DisplayName string `json:"displayName"`
			Hidden      bool   `json:"hidden"`
			IsDefault   bool   `json:"isDefault"`
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
		models = append(models, AgentModel{Name: name, Title: one.DisplayName, IsDefault: one.IsDefault})
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
