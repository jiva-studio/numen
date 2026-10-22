package settings

import (
	"encoding/json"
	"slices"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The groups the models of a program stand on.
const (
	groupInFull     = "In full"
	groupConfigured = "Named in the settings"
)

// AgentModels are the rows the agent section offers a person: the programs the setting can name, and
// the models of each installed program and of the one the settings name. The section stands in the
// file under the key given.
func AgentModels(at string, held Agent, programs []port.AgentProgram) []port.Model {
	useAt := []string{at, "use"}
	models := getProgramModels(useAt, programs)
	for _, one := range programs {
		if one.IsInstalled || one.Name == held.GetUse() {
			modelAt := []string{at, one.Name, "model"}
			models = append(models, getModelsOfProgram(modelAt, one, held.GetProgram(one.Name).Model)...)
		}
	}
	return models
}

// getProgramModels are the programs the agent setting can name, the default one marked, and the
// choice of none.
func getProgramModels(useAt []string, programs []port.AgentProgram) []port.Model {
	byDefault, hasDefault := port.GetDefaultProgram(programs)

	models := make([]port.Model, 0, len(programs)+1)
	for _, one := range programs {
		presence := port.NotFetched
		if one.IsInstalled {
			presence = port.Present
		}
		models = append(models, port.Model{
			Path:      useAt,
			Name:      one.Name,
			Title:     one.Title,
			IsDefault: hasDefault && one.Name == byDefault.Name,
			Writes:    []port.Setting{newSetting(useAt, one.Name)},
			Presence:  presence,
		})
	}
	return append(models, port.Model{
		Path:      useAt,
		Title:     "Nothing answers",
		IsDefault: !hasDefault,
		Writes:    []port.Setting{newSetting(useAt, "")},
		Presence:  port.NothingToFetch,
	})
}

// getModelsOfProgram are the rows of one program: what it reported, or the model the settings name
// alone where it reported none.
func getModelsOfProgram(at []string, program port.AgentProgram, set string) []port.Model {
	var models []port.Model
	switch {
	case len(program.Models) == 0 && set != "":
		models = []port.Model{{Path: at, Name: set, Title: set, IsDefault: true}}
	case len(program.Models) == 0:
		models = []port.Model{{Path: at, Title: "Whatever this machine answers with", IsDefault: true}}
	default:
		models = getReportedModels(at, set, program.Models)
	}
	for place := range models {
		models[place].Presence = port.NothingToFetch
		models[place].Writes = []port.Setting{newSetting(at, models[place].Name)}
	}
	return models
}

// getReportedModels are the rows of the models a program reported, the one the settings name last
// where it is not among them.
func getReportedModels(at []string, set string, offered []port.AgentModel) []port.Model {
	isSet := slices.ContainsFunc(offered, func(one port.AgentModel) bool { return one.Name == set })
	isReportedDefault := slices.ContainsFunc(offered, func(one port.AgentModel) bool { return one.IsDefault })
	var models []port.Model
	for place, one := range offered {
		title := one.Title
		if title == "" {
			title = one.Name
		}
		isDefault := one.Name == set
		if !isSet {
			isDefault = one.IsDefault || (place == 0 && !isReportedDefault)
		}
		models = append(models, port.Model{
			Path: at, Name: one.Name, Title: title, Group: groupInFull, IsDefault: isDefault,
			ReasoningLevels: one.ReasoningLevels, DefaultReasoning: one.DefaultReasoning,
		})
	}
	if set != "" && !isSet {
		models = append(models, port.Model{Path: at, Name: set, Title: set, Group: groupConfigured})
	}
	return models
}

// newSetting is one setting written down.
func newSetting(at []string, value any) port.Setting {
	response, err := json.Marshal(value)
	if err != nil {
		return port.Setting{Path: at, JSON: ""}
	}
	return port.Setting{Path: at, JSON: string(response)}
}
