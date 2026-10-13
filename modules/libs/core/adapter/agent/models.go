package agent

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The groups the models stand on.
const (
	groupInFull     = "In full"
	groupConfigured = "Named in the settings"
)

// UseAt is where the program that answers stands inside this adapter's
// section of the settings.
var UseAt = []string{"use"}

// GetModels are the models of every installed agent, and of the one the settings
// name. An agent reports its own, once per session. One that reports none
// offers the model its settings name, alone.
func GetModels(held Config) []port.Model {
	return getModels(session, getListed(held))
}

// RefreshModels drops what the agents reported, so the next read asks them again.
func RefreshModels() { session.clear() }

// listed is one agent whose models are on offer.
type listed struct {
	use string
	at  []string
	// set is the model its settings name.
	set string
	// key changes whenever what the agent is asked through changes.
	key    string
	source modelSource
}

// getListed are the agents whose models are on offer: the installed ones, and
// the one the settings name.
func getListed(held Config) []listed {
	apiKey := os.Getenv(anthropicKeyVar)
	all := []struct {
		isInstalled bool
		agent       listed
	}{
		{IsClaudeInstalled(), listed{
			use: UseClaude, at: ModelAt, set: held.Claude.Model, key: apiKey,
			source: claudeModels{key: apiKey, base: os.Getenv(anthropicBaseVar)},
		}},
		{IsAntigravityInstalled(), listed{
			use: UseAntigravity, at: []string{"antigravity", "model"}, set: held.Antigravity.Model,
			source: antigravityModels{},
		}},
		{IsCodexInstalled(), listed{
			use: UseCodex, at: []string{"codex", "model"}, set: held.Codex.Model,
			key: strings.Join(held.Codex.Command, " "), source: codexModels{command: held.Codex.Command},
		}},
	}
	var agents []listed
	for _, one := range all {
		if one.isInstalled || one.agent.use == held.Use {
			agents = append(agents, one.agent)
		}
	}
	return agents
}

// getModels are the rows of these agents, each asked for its models at the same time.
func getModels(kept *catalogue, agents []listed) []port.Model {
	offered := make([][]AgentModel, len(agents))
	var group sync.WaitGroup
	for at, one := range agents {
		group.Add(1)
		go func() {
			defer group.Done()
			offered[at] = kept.get(one.use, one.key, one.source)
		}()
	}
	group.Wait()

	var models []port.Model
	for at, one := range agents {
		models = append(models, getAgentModels(one.at, one.set, offered[at])...)
	}
	return models
}

// getAgentModels are the rows of one agent: what it reported, or the model the
// settings name alone where it reported none.
func getAgentModels(at []string, set string, offered []AgentModel) []port.Model {
	var models []port.Model
	switch {
	case len(offered) == 0 && set != "":
		models = []port.Model{{Path: at, Name: set, Title: set, IsDefault: true}}
	case len(offered) == 0:
		models = []port.Model{{Path: at, Title: "Whatever this machine answers with", IsDefault: true}}
	default:
		models = getReportedModels(at, set, offered)
	}
	for place := range models {
		models[place].Presence = port.NothingToFetch
		models[place].Writes = []port.Setting{newSetting(at, models[place].Name)}
	}
	return models
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

// newSetting is one setting written down.
func newSetting(at []string, value any) port.Setting {
	said, err := json.Marshal(value)
	if err != nil {
		return port.Setting{Path: at, JSON: ""}
	}
	return port.Setting{Path: at, JSON: string(said)}
}

// getReportedModels are the rows of the models an agent reported, the one the settings name
// last where it is not among them.
func getReportedModels(at []string, set string, offered []AgentModel) []port.Model {
	isSet := slices.ContainsFunc(offered, func(one AgentModel) bool { return one.Name == set })
	isReportedDefault := slices.ContainsFunc(offered, func(one AgentModel) bool { return one.IsDefault })
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
		})
	}
	if set != "" && !isSet {
		models = append(models, port.Model{Path: at, Name: set, Title: set, Group: groupConfigured})
	}
	return models
}
