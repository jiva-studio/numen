package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	anthropicKeyVar  = "ANTHROPIC_API_KEY"
	anthropicBaseVar = "ANTHROPIC_BASE_URL"
	anthropicHost    = "https://api.anthropic.com"
)

type AgentModel struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

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
