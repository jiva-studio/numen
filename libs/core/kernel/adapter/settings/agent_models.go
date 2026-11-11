package settings

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// AgentModels returns the rows of the agent section: the programs the setting
// can name, each filed under the given key.
func AgentModels(targetPath string, target Agent, programs []agent.Program) []settingsdomain.Model {
	return settingsdomain.ListAgentModels(
		targetPath,
		target.GetUse(),
		func(name string) string { return target.GetProgram(name).Model },
		programs,
		func(pos []string, value string) settingsdomain.Setting { return newSetting(pos, value) },
	)
}

func newSetting(pos []string, value any) settingsdomain.Setting {
	response, err := json.Marshal(value)
	if err != nil {
		return settingsdomain.Setting{Path: pos, JSON: ""}
	}
	return settingsdomain.Setting{Path: pos, JSON: string(response)}
}
