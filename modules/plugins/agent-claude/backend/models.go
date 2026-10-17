package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
)

const (
	anthropicKeyVar  = "ANTHROPIC_API_KEY"
	anthropicBaseVar = "ANTHROPIC_BASE_URL"
	anthropicHost    = "https://api.anthropic.com"
)

// AgentModel is one model the endpoint lists.
type AgentModel struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

// modelCatalogue keeps the models the endpoint listed, for as long as the key and the endpoint it
// was asked with stay the same. A failure is kept as none.
type modelCatalogue struct {
	mu     sync.Mutex
	key    string
	base   string
	kept   []AgentModel
	isKept bool
}

// set changes the key and the endpoint the models are asked with, and drops what was kept.
func (c *modelCatalogue) set(key, base string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.key, c.base = key, base
	c.kept, c.isKept = nil, false
}

// get are the models the endpoint lists: the ones kept, or those it lists now.
func (c *modelCatalogue) get(ctx context.Context) []AgentModel {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isKept {
		return c.kept
	}
	models, err := fetchAnthropicModels(ctx, c.key, c.base)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		models = nil
	}
	c.kept, c.isKept = models, true
	return c.kept
}

// fetchAnthropicModels asks the endpoint for its models, and answers none where there is no key.
func fetchAnthropicModels(ctx context.Context, key, base string) ([]AgentModel, error) {
	if key == "" {
		key = os.Getenv(anthropicKeyVar)
	}
	if key == "" {
		return nil, nil
	}
	if base == "" {
		base = os.Getenv(anthropicBaseVar)
	}
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = anthropicHost
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/models?limit=1000", nil)
	if err != nil {
		return nil, fmt.Errorf("create models request: %w", err)
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models endpoint answered %s", resp.Status)
	}

	var listing struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}

	var models []AgentModel
	for _, one := range listing.Data {
		if one.ID != "" {
			models = append(models, AgentModel{Name: one.ID, Title: one.DisplayName})
		}
	}
	return models, nil
}
