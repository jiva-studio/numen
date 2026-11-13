// Package reasoning knows which levels of reasoning a model of a connection type takes.
package reasoning

import (
	"slices"
	"strings"
)

const (
	typeOpenRouter = "openrouter"
	typeOpenAI     = "openai"
)

// Level is a level of reasoning a model takes.
type Level struct {
	Value string
	Title string
}

// GetLevels returns the levels the model takes, or none. OpenRouter takes them for every model; OpenAI
// for its reasoning models.
func GetLevels(typeID, model string) []Level {
	switch typeID {
	case typeOpenRouter:
		return listLevels()
	case typeOpenAI:
		name := strings.ToLower(strings.TrimSpace(model))
		for _, prefix := range []string{"o1", "o3", "o4", "gpt-5"} {
			if strings.HasPrefix(name, prefix) {
				return listLevels()
			}
		}
	}
	return nil
}

// IsSupported reports whether the model takes the level. An empty level is the default of the model,
// which every model takes.
func IsSupported(typeID, model, level string) bool {
	if level == "" {
		return true
	}
	return slices.ContainsFunc(GetLevels(typeID, model), func(one Level) bool { return one.Value == level })
}

func listLevels() []Level {
	return []Level{{Value: "low", Title: "Low"}, {Value: "medium", Title: "Medium"}, {Value: "high", Title: "High"}}
}
