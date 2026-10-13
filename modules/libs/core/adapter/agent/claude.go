package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ModelAt is where the model the agent answers with stands inside this
// adapter's section of the settings.
var ModelAt = []string{"claude", "model"}

const (
	anthropicKeyVar  = "ANTHROPIC_API_KEY"
	anthropicBaseVar = "ANTHROPIC_BASE_URL"
	anthropicHost    = "https://api.anthropic.com"
)

// claudeModels is the Anthropic models endpoint, asked with the key this machine holds.
type claudeModels struct {
	key  string
	base string
}

// GetModels are the models the endpoint lists, and none where there is no key.
func (s claudeModels) GetModels(ctx context.Context) ([]AgentModel, error) {
	if s.key == "" {
		return nil, nil
	}
	base := strings.TrimRight(s.base, "/")
	if base == "" {
		base = anthropicHost
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/models?limit=1000", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("x-api-key", s.key)
	request.Header.Set("anthropic-version", "2023-06-01")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models endpoint answered %s", response.Status)
	}
	var listing struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listing); err != nil {
		return nil, err
	}
	var models []AgentModel
	for _, one := range listing.Data {
		if one.ID != "" {
			models = append(models, AgentModel{Name: one.ID, Title: one.DisplayName})
		}
	}
	return models, nil
}
