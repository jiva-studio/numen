package agent

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The shelves the models stand on.
const (
	shelfSize       = "By how large it is"
	shelfInFull     = "In full"
	shelfConfigured = "Named in the settings"
)

// ModelAt is where the model the agent answers with stands inside this
// adapter's section of the settings, and UseAt the program that answers.
var (
	ModelAt = []string{"claude", "model"}
	UseAt   = []string{"use"}
)

// GetModels are the models the agent can answer with.
func GetModels(held Config) []port.Model {
	switch held.Use {
	case UseAntigravity:
		modelAt := []string{"antigravity", "model"}
		models := []port.Model{
			{Path: modelAt, Title: "Default (Gemini 3.7 Flash)", Name: "gemini-3.7-flash", IsDefault: true, Presence: port.NothingToFetch},
			{Path: modelAt, Name: "gemini-2.5-pro", Title: "Gemini 2.5 Pro", Shelf: shelfInFull, Presence: port.NothingToFetch},
			{Path: modelAt, Name: "gemini-2.5-flash", Title: "Gemini 2.5 Flash", Shelf: shelfInFull, Presence: port.NothingToFetch},
		}
		if name := held.Antigravity.Model; name != "" && !hasModel(models, name) {
			models = append(models, port.Model{Path: modelAt, Name: name, Title: name, Shelf: shelfConfigured, Presence: port.NothingToFetch})
		}
		for at := range models {
			models[at].Writes = []port.Setting{newSetting(modelAt, models[at].Name)}
		}
		return models

	case UseCodex:
		return getCodexModels(held)

	default:
		models := []port.Model{{
			Path:      ModelAt,
			Title:     "Whatever this machine answers with",
			IsDefault: true,
			Presence:  port.NothingToFetch,
		}}
		for _, one := range []string{"opus", "sonnet", "haiku"} {
			models = append(models, port.Model{
				Path: ModelAt, Name: one, Title: one, Shelf: shelfSize, Presence: port.NothingToFetch,
			})
		}
		for _, one := range []string{"claude-opus-5", "claude-sonnet-5", "claude-haiku-4-5"} {
			models = append(models, port.Model{
				Path: ModelAt, Name: one, Title: one, Shelf: shelfInFull, Presence: port.NothingToFetch,
			})
		}
		if name := held.Claude.Model; name != "" && !hasModel(models, name) {
			models = append(models, port.Model{
				Path: ModelAt, Name: name, Title: name, Shelf: shelfConfigured, Presence: port.NothingToFetch,
			})
		}
		for at := range models {
			models[at].Writes = []port.Setting{newSetting(ModelAt, models[at].Name)}
		}
		return models
	}
}

// GetPrograms are the programs the agent setting can name.
func GetPrograms() []port.Model {
	claudePresence := port.NotFetched
	if IsClaudeInstalled() {
		claudePresence = port.Present
	}
	agyPresence := port.NotFetched
	if IsAntigravityInstalled() {
		agyPresence = port.Present
	}
	codexPresence := port.NotFetched
	if IsCodexInstalled() {
		codexPresence = port.Present
	}

	return []port.Model{
		{
			Path:      UseAt,
			Name:      UseClaude,
			Title:     "Claude Code",
			IsDefault: true,
			Writes:    []port.Setting{newSetting(UseAt, UseClaude)},
			Presence:  claudePresence,
		},
		{
			Path:     UseAt,
			Name:     UseAntigravity,
			Title:    "Antigravity",
			Writes:   []port.Setting{newSetting(UseAt, UseAntigravity)},
			Presence: agyPresence,
		},
		{
			Path:     UseAt,
			Name:     UseCodex,
			Title:    "OpenAI Codex",
			Writes:   []port.Setting{newSetting(UseAt, UseCodex)},
			Presence: codexPresence,
		},
		{
			Path:     UseAt,
			Title:    "Nothing answers",
			Writes:   []port.Setting{newSetting(UseAt, "")},
			Presence: port.NothingToFetch,
		},
	}
}

// hasModel says whether one of these models is already the name given.
func hasModel(models []port.Model, name string) bool {
	for _, one := range models {
		if one.Name == name {
			return true
		}
	}
	return false
}

// newSetting is one setting written down.
func newSetting(at []string, value any) port.Setting {
	said, err := json.Marshal(value)
	if err != nil {
		return port.Setting{Path: at, JSON: ""}
	}
	return port.Setting{Path: at, JSON: string(said)}
}

type codexCache struct {
	Models []struct {
		Slug        string `json:"slug"`
		DisplayName string `json:"display_name"`
		Visibility  string `json:"visibility"`
		Priority    int    `json:"priority"`
	} `json:"models"`
}

func getCodexModels(held Config) []port.Model {
	modelAt := []string{"codex", "model"}
	var models []port.Model

	if home, err := os.UserHomeDir(); err == nil {
		cachePath := filepath.Join(home, ".codex", "models_cache.json")
		if data, err := os.ReadFile(cachePath); err == nil {
			var cache codexCache
			if err := json.Unmarshal(data, &cache); err == nil {
				for _, m := range cache.Models {
					if m.Visibility != "hide" && m.Slug != "" {
						title := m.DisplayName
						if title == "" {
							title = m.Slug
						}
						models = append(models, port.Model{
							Path:     modelAt,
							Name:     m.Slug,
							Title:    title,
							Shelf:    shelfInFull,
							Presence: port.NothingToFetch,
						})
					}
				}
			}
		}
	}

	if len(models) == 0 {
		models = []port.Model{
			{Path: modelAt, Title: "Default (GPT-5.6 Terra)", Name: "gpt-5.6-terra", IsDefault: true, Presence: port.NothingToFetch},
			{Path: modelAt, Name: "gpt-5.6-luna", Title: "GPT-5.6 Luna", Shelf: shelfInFull, Presence: port.NothingToFetch},
		}
	} else {
		models[0].IsDefault = true
		models[0].Title = "Default (" + models[0].Title + ")"
	}

	if name := held.Codex.Model; name != "" && !hasModel(models, name) {
		models = append(models, port.Model{Path: modelAt, Name: name, Title: name, Shelf: shelfConfigured, Presence: port.NothingToFetch})
	}
	for at := range models {
		models[at].Writes = []port.Setting{newSetting(modelAt, models[at].Name)}
	}
	return models
}
