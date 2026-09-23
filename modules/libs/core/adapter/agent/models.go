package agent

import (
	"encoding/json"

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

// GetModels are the models the agent can answer with. The command line takes a
// size on its own and a name in full, and an installation naming neither
// answers with whatever it is set up to answer with.
//
// The model is reached where the agent runs, and nothing of it is fetched here.
// Every place named here stands inside this section, and whoever composes the
// file says where the section is.
func GetModels(held Config) []port.Model {
	models := []port.Model{{
		Path:      ModelAt,
		Title:     "Whatever this machine answers with",
		IsDefault: true,
	}}
	for _, one := range []string{"opus", "sonnet", "haiku"} {
		models = append(models, port.Model{
			Path: ModelAt, Name: one, Title: one, Shelf: shelfSize,
		})
	}
	for _, one := range []string{"claude-opus-5", "claude-sonnet-5", "claude-haiku-4-5"} {
		models = append(models, port.Model{
			Path: ModelAt, Name: one, Title: one, Shelf: shelfInFull,
		})
	}
	if name := held.Claude.Model; name != "" && !hasModel(models, name) {
		models = append(models, port.Model{
			Path: ModelAt, Name: name, Title: name, Shelf: shelfConfigured,
		})
	}
	for at := range models {
		models[at].Writes = []port.Setting{newSetting(ModelAt, models[at].Name)}
	}
	return models
}

// GetPrograms are the programs the agent setting can name. This build reaches
// the one it has an adapter for, and a name it cannot reach is not offered.
func GetPrograms() []port.Model {
	return []port.Model{
		{
			Path:      UseAt,
			Name:      UseClaude,
			Title:     "Claude Code",
			IsDefault: true,
			Writes:    []port.Setting{newSetting(UseAt, UseClaude)},
		},
		{
			Path:   UseAt,
			Title:  "Nothing answers",
			Writes: []port.Setting{newSetting(UseAt, "")},
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
