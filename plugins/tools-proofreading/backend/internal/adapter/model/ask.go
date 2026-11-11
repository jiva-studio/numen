// Package model asks the language model, through the host or over HTTP, and holds the mock proofreader.
package model

import (
	"context"
	"os"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/ai"
)

// EnvMockML names the variable that has the plugin answer every question with the mock proofreader.
const EnvMockML = "NUMEN_MOCK_ML"

const mockEnabled = "true"

// purpose is what every question to the model is for. It routes nothing.
const purpose = "proofreading"

// Completer sends a question to the host and returns the answer.
type Completer func(ctx context.Context, question *ai.Question) (string, error)

// Endpoint is the HTTP chat service a configured provider points at.
type Endpoint struct {
	Provider string
	BaseURL  string
	Model    string
	APIKey   string
}

// DefaultEndpoint has the host answer every question.
func DefaultEndpoint() Endpoint {
	return Endpoint{Provider: "agent", BaseURL: "", Model: defaultModel, APIKey: ""}
}

// IsMockEnabled reports whether the value of EnvMockML selects the mock model.
func IsMockEnabled(value string) bool { return value == mockEnabled }

// NewAsk returns the function that asks the model one question. A provider that is openrouter,
// openai or custom is asked over HTTP by the plugin itself, unless the mock is enabled in the
// environment; every other provider is asked through complete. A nil getEndpoint asks complete.
func NewAsk(complete Completer, getEndpoint func() Endpoint) func(ctx context.Context, instruction, prompt string) (string, error) {
	return func(ctx context.Context, instruction, prompt string) (string, error) {
		if getEndpoint != nil && !IsMockSelected() {
			endpoint := getEndpoint()
			if endpoint.Provider == "openrouter" || endpoint.Provider == "openai" || endpoint.Provider == "custom" {
				return askDirectAPI(ctx, endpoint, instruction, prompt)
			}
		}
		return complete(ctx, &ai.Question{
			Purpose: purpose,
			Messages: []ai.Message{
				{Role: "system", Text: instruction},
				{Role: "user", Text: prompt},
			},
		})
	}
}

// IsMockSelected reports whether the environment of the process selects the mock model.
func IsMockSelected() bool { return IsMockEnabled(os.Getenv(EnvMockML)) }

// WithSettings returns the endpoint with the provider, base URL and model the settings name. A
// provider or a model the settings leave empty stays as it is.
func (e Endpoint) WithSettings(values map[string]string) Endpoint {
	if provider := values["provider"]; provider != "" {
		e.Provider = provider
	}
	if baseURL, ok := values["baseUrl"]; ok {
		e.BaseURL = baseURL
	}
	if name := values["model"]; name != "" {
		e.Model = name
	}
	return e
}
