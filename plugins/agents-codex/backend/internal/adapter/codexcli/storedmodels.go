package codexcli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/plugins/agents-codex/backend/agentmodel"
)

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

func readCodexCache() []agentmodel.Model {
	home, homeErr := os.UserHomeDir()
	data, readErr := os.ReadFile(filepath.Join(home, ".codex", "models_cache.json"))
	var cache codexCache
	if errors.Join(homeErr, readErr, json.Unmarshal(data, &cache)) != nil {
		return nil
	}
	var models []agentmodel.Model
	for _, m := range cache.Models {
		if m.Visibility == "hide" || m.Slug == "" {
			continue
		}
		var levels []string
		for _, one := range m.Supported {
			levels = append(levels, one.Effort)
		}
		models = append(models, agentmodel.New(m.Slug, m.DisplayName, false, levels, m.Default))
	}
	return models
}
