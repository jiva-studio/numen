package settings

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
)

// The groups and titles the models of an agent program stand on.
const (
	GroupInFull            = "In full"
	GroupConfigured        = "Named in the settings"
	DefaultAgentModelTitle = "Default model"
	NoAgentTitle           = "No agent responds"
)

// ListAgentModels returns the rows of the agent section: the programs the setting can name, each
// filed under the given key.
func ListAgentModels(
	targetPath, selectedUse string,
	getProgramModel func(string) string,
	programs []agent.Program,
	formatWrite func(path []string, value string) Setting,
) []Model {
	useAt := []string{targetPath, "use"}
	models := getProgramModels(useAt, programs, formatWrite)
	for _, one := range programs {
		if one.IsInstalled || one.Name == selectedUse {
			modelAt := []string{targetPath, one.Name, "model"}
			models = append(models, getModelsOfProgram(modelAt, one, getProgramModel(one.Name), formatWrite)...)
		}
	}
	return models
}

func getProgramModels(
	useAt []string, programs []agent.Program, formatWrite func(path []string, value string) Setting,
) []Model {
	byDefault, hasDefault := agent.GetDefaultProgram(programs)

	models := make([]Model, 0, len(programs)+1)
	for _, one := range programs {
		presence := NotFetched
		if one.IsInstalled {
			presence = Present
		}
		models = append(models, Model{
			Path:             useAt,
			Name:             one.Name,
			Title:            one.Title,
			Group:            "",
			IsDefault:        hasDefault && one.Name == byDefault.Name,
			Writes:           []Setting{formatWrite(useAt, one.Name)},
			Presence:         presence,
			ReasoningLevels:  nil,
			DefaultReasoning: "",
		})
	}
	return append(models, Model{
		Path:             useAt,
		Name:             "",
		Title:            NoAgentTitle,
		Group:            "",
		IsDefault:        !hasDefault,
		Writes:           []Setting{formatWrite(useAt, "")},
		Presence:         NothingToFetch,
		ReasoningLevels:  nil,
		DefaultReasoning: "",
	})
}

func getModelsOfProgram(
	pos []string, program agent.Program, set string, formatWrite func(path []string, value string) Setting,
) []Model {
	var models []Model
	switch {
	case len(program.Models) == 0 && set != "":
		models = []Model{newDefaultModel(pos, set, set)}
	case len(program.Models) == 0:
		models = []Model{newDefaultModel(pos, "", DefaultAgentModelTitle)}
	default:
		models = getReportedModels(pos, set, program.Models)
	}
	for place := range models {
		models[place].Presence = NothingToFetch
		models[place].Writes = []Setting{formatWrite(pos, models[place].Name)}
	}
	return models
}

func getReportedModels(pos []string, set string, available []agent.Model) []Model {
	isSet := slices.ContainsFunc(available, func(one agent.Model) bool { return one.Name == set })
	isReportedDefault := slices.ContainsFunc(available, func(one agent.Model) bool { return one.IsDefault })
	var models []Model
	for place, one := range available {
		title := one.Title
		if title == "" {
			title = one.Name
		}
		isDefault := one.Name == set
		if !isSet {
			isDefault = one.IsDefault || (place == 0 && !isReportedDefault)
		}
		models = append(models, Model{
			Path: pos, Name: one.Name, Title: title, Group: GroupInFull, IsDefault: isDefault,
			Writes: nil, Presence: NothingToFetch,
			ReasoningLevels: one.ReasoningLevels, DefaultReasoning: one.DefaultReasoning,
		})
	}
	if set != "" && !isSet {
		models = append(models, Model{
			Path: pos, Name: set, Title: set, Group: GroupConfigured, IsDefault: false,
			Writes: nil, Presence: NothingToFetch, ReasoningLevels: nil, DefaultReasoning: "",
		})
	}
	return models
}

func newDefaultModel(pos []string, name, title string) Model {
	return Model{
		Path: pos, Name: name, Title: title, Group: "", IsDefault: true,
		Writes: nil, Presence: NothingToFetch, ReasoningLevels: nil, DefaultReasoning: "",
	}
}
