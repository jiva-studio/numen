package agent

import "github.com/jiva-studio/numen/modules/libs/core/port"

func getAntigravityModels(held Config) []port.Model {
	modelAt := []string{"antigravity", "model"}
	models := []port.Model{
		{Path: modelAt, Title: "Default (Gemini 3.7 Flash)", Name: "gemini-3.7-flash", IsDefault: true, Presence: port.NothingToFetch},
		{Path: modelAt, Name: "gemini-2.5-pro", Title: "Gemini 2.5 Pro", Group: groupInFull, Presence: port.NothingToFetch},
		{Path: modelAt, Name: "gemini-2.5-flash", Title: "Gemini 2.5 Flash", Group: groupInFull, Presence: port.NothingToFetch},
	}
	if name := held.Antigravity.Model; name != "" && !hasModel(models, name) {
		models = append(models, port.Model{Path: modelAt, Name: name, Title: name, Group: groupConfigured, Presence: port.NothingToFetch})
	}
	for at := range models {
		models[at].Writes = []port.Setting{newSetting(modelAt, models[at].Name)}
	}
	return models
}
