package agent

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
)

type codexCache struct {
	Models []struct {
		Slug        string `json:"slug"`
		DisplayName string `json:"display_name"`
		Visibility  string `json:"visibility"`
	} `json:"models"`
}

// readCodexCache is the visible models in the cache the codex program keeps in
// the home folder, and none where there is no readable cache.
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

// codexModels is `codex app-server` asked model/list, and its cache read where it cannot answer.
type codexModels struct {
	command []string
}

// GetModels are the models the app server lists, else those in the cache of the codex program.
func (s codexModels) GetModels(ctx context.Context) ([]AgentModel, error) {
	models, err := s.ask(ctx)
	if len(models) > 0 {
		return models, nil
	}
	if cached := readCodexCache(); len(cached) > 0 {
		return cached, nil
	}
	return nil, err
}

// ask starts the app server, lists the models and stops it again.
func (s codexModels) ask(ctx context.Context) ([]AgentModel, error) {
	name, args := FindCodex(), []string(nil)
	if len(s.command) > 0 {
		name, args = s.command[0], s.command[1:]
	}
	running, stop := context.WithCancel(ctx)
	defer stop()
	cmd := exec.CommandContext(running, name, slices.Concat(args, []string{"app-server"})...)
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
		in.Close()
		stop()
		_ = cmd.Wait()
	}()
	return listCodexModels(in, out)
}

// listCodexModels does the handshake of the app server over its lines of JSON, and asks model/list.
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

// writeCodexLine writes one message as a line of JSON.
func writeCodexLine(in io.Writer, message map[string]any) error {
	raw, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = in.Write(append(raw, '\n'))
	return err
}

// readCodexAnswer is the result of the answer to the request with this id, read past anything else
// the server says.
func readCodexAnswer(lines *bufio.Scanner, id int) (json.RawMessage, error) {
	for lines.Scan() {
		var message struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(lines.Bytes(), &message); err != nil || message.ID == nil || *message.ID != id {
			continue
		}
		if message.Error != nil {
			return nil, fmt.Errorf("app server refused request %d: %s", id, message.Error.Message)
		}
		return message.Result, nil
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("app server ended before it answered")
}
