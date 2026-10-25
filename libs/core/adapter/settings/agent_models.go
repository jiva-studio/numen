package settings

import (
	"encoding/json"
	"slices"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// The groups the models of a program stand on.
const (
	groupInFull     = "In full"
	groupConfigured = "Named in the settings"
)

// AgentModels are the rows the agent section offers a person: the programs the setting can name, and
// file under the key given.
func AgentModels(targetPath string, target Agent, programs []agentport.AgentProgram) []settingsport.Model {
	useAt := []string{targetPath, "use"}
	models := getProgramModels(useAt, programs)
	for _, one := range programs {
		if one.IsInstalled || one.Name == target.GetUse() {
			modelAt := []string{targetPath, one.Name, "model"}
			models = append(models, getModelsOfProgram(modelAt, one, target.GetProgram(one.Name).Model)...)
		}
	}
	return models
}

// getProgramModels are the programs the agent setting can name, the default one marked, and the
// choice of none.
func getProgramModels(useAt []string, programs []agentport.AgentProgram) []settingsport.Model {
	byDefault, hasDefault := agentport.GetDefaultProgram(programs)

	models := make([]settingsport.Model, 0, len(programs)+1)
	for _, one := range programs {
		presence := settingsport.NotFetched
		if one.IsInstalled {
			presence = settingsport.Present
		}
		models = append(models, settingsport.Model{
			Path:      useAt,
			Name:      one.Name,
			Title:     one.Title,
			IsDefault: hasDefault && one.Name == byDefault.Name,
			Writes:    []settingsport.Setting{newSetting(useAt, one.Name)},
			Presence:  presence,
		})
	}
	return append(models, settingsport.Model{
		Path:      useAt,
		Title:     "Nothing answers",
		IsDefault: !hasDefault,
		Writes:    []settingsport.Setting{newSetting(useAt, "")},
		Presence:  settingsport.NothingToFetch,
	})
}

// getModelsOfProgram are the rows of one program: what it reported, or the model the settings name
// alone where it reported none.
func getModelsOfProgram(pos []string, program agentport.AgentProgram, set string) []settingsport.Model {
	var models []settingsport.Model
	switch {
	case len(program.Models) == 0 && set != "":
		models = []settingsport.Model{{Path: pos, Name: set, Title: set, IsDefault: true}}
	case len(program.Models) == 0:
		models = []settingsport.Model{{Path: pos, Title: "Whatever this machine answers with", IsDefault: true}}
	default:
		models = getReportedModels(pos, set, program.Models)
	}
	for place := range models {
		models[place].Presence = settingsport.NothingToFetch
		models[place].Writes = []settingsport.Setting{newSetting(pos, models[place].Name)}
	}
	return models
}

// getReportedModels are the rows of the models a program reported, the one the settings name last
// where it is not among them.
func getReportedModels(pos []string, set string, available []agentport.AgentModel) []settingsport.Model {
	isSet := slices.ContainsFunc(available, func(one agentport.AgentModel) bool { return one.Name == set })
	isReportedDefault := slices.ContainsFunc(available, func(one agentport.AgentModel) bool { return one.IsDefault })
	var models []settingsport.Model
	for place, one := range available {
		title := one.Title
		if title == "" {
			title = one.Name
		}
		isDefault := one.Name == set
		if !isSet {
			isDefault = one.IsDefault || (place == 0 && !isReportedDefault)
		}
		models = append(models, settingsport.Model{
			Path: pos, Name: one.Name, Title: title, Group: groupInFull, IsDefault: isDefault,
			ReasoningLevels: one.ReasoningLevels, DefaultReasoning: one.DefaultReasoning,
		})
	}
	if set != "" && !isSet {
		models = append(models, settingsport.Model{Path: pos, Name: set, Title: set, Group: groupConfigured})
	}
	return models
}

// newSetting is one setting written down.
func newSetting(pos []string, value any) settingsport.Setting {
	response, err := json.Marshal(value)
	if err != nil {
		return settingsport.Setting{Path: pos, JSON: ""}
	}
	return settingsport.Setting{Path: pos, JSON: string(response)}
}
