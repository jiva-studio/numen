package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
)

const (
	anthropicKeyVar  = "ANTHROPIC_API_KEY"
	anthropicBaseVar = "ANTHROPIC_BASE_URL"
	anthropicHost    = "https://api.anthropic.com"
)

// modelsLimit is how long the endpoint has to list its models.
const modelsLimit = 15 * time.Second

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

// update changes the key and the endpoint the models are asked with, for each the values give, and
// drops what was kept.
func (c *modelCatalogue) update(values map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if key, ok := server.ReadSecret(values, server.SecretAPIKey); ok {
		c.key = key
	}
	if base, ok := values["baseUrl"]; ok {
		c.base = base
	}
	c.kept, c.isKept = nil, false
}

// credentials are the key and the endpoint the models are asked with.
func (c *modelCatalogue) credentials() (key, base string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.key, c.base
}

// get are the models the endpoint lists: the ones kept, or those it lists now.
func (c *modelCatalogue) get(ctx context.Context) []AgentModel {
	c.mu.Lock()
	if c.isKept {
		kept := c.kept
		c.mu.Unlock()
		return kept
	}
	key, base := c.key, c.base
	c.mu.Unlock()

	asking, stop := context.WithTimeout(ctx, modelsLimit)
	defer stop()
	models, err := fetchAnthropicModels(asking, key, base)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		models = nil
	}

	// What was listed under a key and an endpoint that have been changed since is not kept.
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.key == key && c.base == base {
		c.kept, c.isKept = models, true
	}
	return models
}

// fetchAnthropicModels asks the endpoint for its models, and answers none where there is no key.
func fetchAnthropicModels(ctx context.Context, key, base string) ([]AgentModel, error) {
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
