package acp

import "github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"

// Model is one model an ACP agent reports for its sessions.
type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Listing is what an agent reports for a session: its models and the one it answers with.
type Listing struct {
	Models  []Model           `json:"models"`
	Current string            `json:"current"`
	Levels  []reasoning.Level `json:"levels,omitempty"`
	Level   string            `json:"level,omitempty"`
}

// configOption is one session config option. A select option's values are listed alone or in
// groups.
type configOption struct {
	ID           string        `json:"id"`
	Category     string        `json:"category"`
	Type         string        `json:"type"`
	CurrentValue string        `json:"currentValue"`
	Options      []optionValue `json:"options"`
}

type optionValue struct {
	Value   string        `json:"value"`
	Name    string        `json:"name"`
	Options []optionValue `json:"options"`
}

func (o configOption) getModels() []Model {
	var models []Model
	var walk func([]optionValue)
	walk = func(values []optionValue) {
		for _, one := range values {
			if one.Value != "" {
				models = append(models, Model{ID: one.Value, Name: one.Name})
			}
			walk(one.Options)
		}
	}
	walk(o.Options)
	return models
}

func (o configOption) getLevels() []reasoning.Level {
	var levels []reasoning.Level
	for _, one := range o.getModels() {
		title := one.Name
		if title == "" {
			title = one.ID
		}
		levels = append(levels, reasoning.Level{Name: one.ID, Title: title})
	}
	return levels
}
