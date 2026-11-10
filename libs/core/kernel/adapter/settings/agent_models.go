package settings

import (
	"encoding/json"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// The groups the models of a program stand on.
const (
	groupInFull     = "In full"
	groupConfigured = "Named in the settings"
)

// AgentModels returns the rows of the agent section: the programs the setting
// can name, each filed under the given key.
func AgentModels(targetPath string, target Agent, programs []agent.Program) []settingsdomain.Model {
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

func getProgramModels(useAt []string, programs []agent.Program) []settingsdomain.Model {
	byDefault, hasDefault := agent.GetDefaultProgram(programs)

	models := make([]settingsdomain.Model, 0, len(programs)+1)
	for _, one := range programs {
		presence := settingsdomain.NotFetched
		if one.IsInstalled {
			presence = settingsdomain.Present
		}
		models = append(models, settingsdomain.Model{
			Path:      useAt,
			Name:      one.Name,
			Title:     one.Title,
			IsDefault: hasDefault && one.Name == byDefault.Name,
			Writes:    []settingsdomain.Setting{newSetting(useAt, one.Name)},
			Presence:  presence,
		})
	}
	return append(models, settingsdomain.Model{
		Path:      useAt,
		Title:     "No agent responds",
		IsDefault: !hasDefault,
		Writes:    []settingsdomain.Setting{newSetting(useAt, "")},
		Presence:  settingsdomain.NothingToFetch,
	})
}

func getModelsOfProgram(pos []string, program agent.Program, set string) []settingsdomain.Model {
	var models []settingsdomain.Model
	switch {
	case len(program.Models) == 0 && set != "":
		models = []settingsdomain.Model{{Path: pos, Name: set, Title: set, IsDefault: true}}
	case len(program.Models) == 0:
		models = []settingsdomain.Model{{Path: pos, Title: "Default model", IsDefault: true}}
	default:
		models = getReportedModels(pos, set, program.Models)
	}
	for place := range models {
		models[place].Presence = settingsdomain.NothingToFetch
		models[place].Writes = []settingsdomain.Setting{newSetting(pos, models[place].Name)}
	}
	return models
}

func getReportedModels(pos []string, set string, available []agent.Model) []settingsdomain.Model {
	isSet := slices.ContainsFunc(available, func(one agent.Model) bool { return one.Name == set })
	isReportedDefault := slices.ContainsFunc(available, func(one agent.Model) bool { return one.IsDefault })
	var models []settingsdomain.Model
	for place, one := range available {
		title := one.Title
		if title == "" {
			title = one.Name
		}
		isDefault := one.Name == set
		if !isSet {
			isDefault = one.IsDefault || (place == 0 && !isReportedDefault)
		}
		models = append(models, settingsdomain.Model{
			Path: pos, Name: one.Name, Title: title, Group: groupInFull, IsDefault: isDefault,
			ReasoningLevels: one.ReasoningLevels, DefaultReasoning: one.DefaultReasoning,
		})
	}
	if set != "" && !isSet {
		models = append(models, settingsdomain.Model{Path: pos, Name: set, Title: set, Group: groupConfigured})
	}
	return models
}

func newSetting(pos []string, value any) settingsdomain.Setting {
	response, err := json.Marshal(value)
	if err != nil {
		return settingsdomain.Setting{Path: pos, JSON: ""}
	}
	return settingsdomain.Setting{Path: pos, JSON: string(response)}
}
