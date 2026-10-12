package agent

import (
	"encoding/json"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The groups the models stand on.
const (
	groupSize       = "By how large it is"
	groupInFull     = "In full"
	groupConfigured = "Named in the settings"
)

// UseAt is where the program that answers stands inside this adapter's
// section of the settings.
var UseAt = []string{"use"}

// GetModels are the models the agent can answer with.
func GetModels(held Config) []port.Model {
	switch held.Use {
	case UseAntigravity:
		return getAntigravityModels(held)

	case UseCodex:
		return getCodexModels(held)

	default:
		return getClaudeModels(held)
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
