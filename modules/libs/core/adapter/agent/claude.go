package agent

import "github.com/jiva-studio/numen/modules/libs/core/port"

// ModelAt is where the model the agent answers with stands inside this
// adapter's section of the settings.
var ModelAt = []string{"claude", "model"}

func getClaudeModels(held Config) []port.Model {
	models := []port.Model{{
		Path:      ModelAt,
		Title:     "Whatever this machine answers with",
		IsDefault: true,
		Presence:  port.NothingToFetch,
	}}
	for _, one := range []string{"opus", "sonnet", "haiku"} {
		models = append(models, port.Model{
			Path: ModelAt, Name: one, Title: one, Group: groupSize, Presence: port.NothingToFetch,
		})
	}
	for _, one := range []string{"claude-opus-5", "claude-sonnet-5", "claude-haiku-4-5"} {
		models = append(models, port.Model{
			Path: ModelAt, Name: one, Title: one, Group: groupInFull, Presence: port.NothingToFetch,
		})
	}
	if name := held.Claude.Model; name != "" && !hasModel(models, name) {
		models = append(models, port.Model{
			Path: ModelAt, Name: name, Title: name, Group: groupConfigured, Presence: port.NothingToFetch,
		})
	}
	for at := range models {
		models[at].Writes = []port.Setting{newSetting(ModelAt, models[at].Name)}
	}
	return models
}
