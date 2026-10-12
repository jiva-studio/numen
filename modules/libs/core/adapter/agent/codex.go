package agent

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

type codexCache struct {
	Models []struct {
		Slug        string `json:"slug"`
		DisplayName string `json:"display_name"`
		Visibility  string `json:"visibility"`
		Priority    int    `json:"priority"`
	} `json:"models"`
}

// readCodexCache is the visible models in the cache the codex program keeps in
// the home folder, and none where there is no readable cache.
func readCodexCache(modelAt []string) []port.Model {
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
	var models []port.Model
	for _, m := range cache.Models {
		if m.Visibility == "hide" || m.Slug == "" {
			continue
		}
		title := m.DisplayName
		if title == "" {
			title = m.Slug
		}
		models = append(models, port.Model{
			Path:     modelAt,
			Name:     m.Slug,
			Title:    title,
			Group:    groupInFull,
			Presence: port.NothingToFetch,
		})
	}
	return models
}

func getCodexModels(held Config) []port.Model {
	modelAt := []string{"codex", "model"}
	models := readCodexCache(modelAt)

	if len(models) == 0 {
		models = []port.Model{
			{Path: modelAt, Title: "Default (GPT-5.6 Terra)", Name: "gpt-5.6-terra", IsDefault: true, Presence: port.NothingToFetch},
			{Path: modelAt, Name: "gpt-5.6-luna", Title: "GPT-5.6 Luna", Group: groupInFull, Presence: port.NothingToFetch},
		}
	} else {
		models[0].IsDefault = true
		models[0].Title = "Default (" + models[0].Title + ")"
	}

	if name := held.Codex.Model; name != "" && !hasModel(models, name) {
		models = append(models, port.Model{Path: modelAt, Name: name, Title: name, Group: groupConfigured, Presence: port.NothingToFetch})
	}
	for at := range models {
		models[at].Writes = []port.Setting{newSetting(modelAt, models[at].Name)}
	}
	return models
}
