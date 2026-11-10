// Package agentmodel holds a model that an agent program offers and the reasoning levels it supports.
package agentmodel

import "github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"

// Model is one model of an agent program. Reasoning lists the supported levels from lowest to
// highest, and DefaultReasoning is the one of them the program uses when none is asked for.
type Model struct {
	Name             string            `json:"name"`
	Title            string            `json:"title"`
	IsDefault        bool              `json:"isDefault,omitempty"`
	Reasoning        []reasoning.Level `json:"-"`
	DefaultReasoning string            `json:"-"`
}

// New creates a model from the effort names a program reports. A name that is empty or listed twice
// is dropped.
func New(id, label string, isPreferred bool, efforts []string, byDefault string) Model {
	var levels []reasoning.Level
	for _, effort := range efforts {
		if level := reasoning.NewLevel(effort); level.Name != "" && !reasoning.Contains(levels, level.Name) {
			levels = append(levels, level)
		}
	}
	reasoning.Sort(levels)
	return Model{
		Name: id, Title: label, IsDefault: isPreferred,
		Reasoning: levels, DefaultReasoning: reasoning.Pick(levels, byDefault),
	}
}
