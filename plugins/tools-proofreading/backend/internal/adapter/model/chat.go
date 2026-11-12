package model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/endpoint"
)

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model    string                  `json:"model"`
	Messages []chatCompletionMessage `json:"messages"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatCompletionMessage `json:"message"`
	} `json:"choices"`
}

func resolveDirectBaseURL(cfg endpoint.Endpoint) string {
	switch cfg.Provider {
	case "openrouter":
		if cfg.BaseURL != "" {
			return cfg.BaseURL
		}
		return "https://openrouter.ai/api/v1"
	case "openai":
		if cfg.BaseURL != "" {
			return cfg.BaseURL
		}
		return "https://api.openai.com/v1"
	default:
		return cfg.BaseURL
	}
}

func askDirectAPI(ctx context.Context, cfg endpoint.Endpoint, instruction, prompt string) (answer string, err error) {
	baseURL := strings.TrimSuffix(resolveDirectBaseURL(cfg), "/")
	if baseURL == "" {
		return "", errors.New("service URL is not configured")
	}
	model := cfg.Model
	if model == "" {
		model = endpoint.DefaultModel
	}
	body, err := json.Marshal(chatCompletionRequest{
		Model: model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: instruction},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	client := &http.Client{
		Transport: nil, CheckRedirect: nil, Jar: nil, Timeout: 2 * time.Minute,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send chat request: %w", err)
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read chat response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("chat API status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}
	var parsed chatCompletionResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("chat API returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}
