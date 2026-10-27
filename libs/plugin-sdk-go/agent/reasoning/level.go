package reasoning

import (
	"slices"
	"strings"

	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// Level is one depth of reasoning.
type Level struct {
	// Name is what a task asks for the level by.
	Name string
	// Title is what the level is called when shown to a person.
	Title string
}

type ReasoningLevel = numenv1.ReasoningLevel

type AgentModel = numenv1.AgentModel

// order lists the level names from the least reasoning to the most.
var order = []string{"minimal", "low", "medium", "high", "xhigh", "max", "ultra"}

var titles = map[string]string{"xhigh": "Extra high"}

// NewLevel is the level of a name, titled in the words a person reads.
func NewLevel(name string) Level {
	name = strings.ToLower(strings.TrimSpace(name))
	if title, found := titles[name]; found {
		return Level{Name: name, Title: title}
	}
	title := name
	if name != "" {
		title = strings.ToUpper(name[:1]) + name[1:]
	}
	return Level{Name: name, Title: title}
}

// NewLevels are the levels of the names, in the order given.
func NewLevels(names ...string) []Level {
	levels := make([]Level, 0, len(names))
	for _, name := range names {
		levels = append(levels, NewLevel(name))
	}
	return levels
}

// IsKnown is whether the name is one of the level names.
func IsKnown(name string) bool {
	return slices.Contains(order, strings.ToLower(strings.TrimSpace(name)))
}

// Contains is whether a level of the name is among the levels.
func Contains(levels []Level, name string) bool {
	return slices.ContainsFunc(levels, func(one Level) bool { return one.Name == name })
}

// Sort orders the levels from the least reasoning to the most, a name outside the known ones last.
func Sort(levels []Level) {
	rank := func(one Level) int {
		if place := slices.Index(order, one.Name); place >= 0 {
			return place
		}
		return len(order)
	}
	slices.SortStableFunc(levels, func(a, b Level) int { return rank(a) - rank(b) })
}

// GetDefault is the level a model works at where a task names none: "medium" where the levels hold
// it, else the first.
func GetDefault(levels []Level) string {
	switch {
	case len(levels) == 0:
		return ""
	case Contains(levels, "medium"):
		return "medium"
	}
	return levels[0].Name
}

// ToProto is the levels in the schema's words.
func ToProto(levels []Level) []*ReasoningLevel {
	if len(levels) == 0 {
		return nil
	}
	out := make([]*ReasoningLevel, 0, len(levels))
	for _, one := range levels {
		out = append(out, &ReasoningLevel{Name: one.Name, Title: one.Title})
	}
	return out
}

// NewModel is a model offered with the levels. The default is empty where it is not one of them.
func NewModel(name, title string, levels []Level, byDefault string) *AgentModel {
	if !Contains(levels, byDefault) {
		byDefault = ""
	}
	return &AgentModel{
		Name: name, Title: title, ReasoningLevels: ToProto(levels), DefaultReasoning: byDefault,
	}
}

// Normalize is the level a task asks for if the levels hold it, else the default of the levels.
func Normalize(levels []Level, requested string) string {
	if Contains(levels, requested) {
		return requested
	}
	return GetDefault(levels)
}

// Pick is the level a task asks for if the levels hold it, else empty.
func Pick(levels []Level, requested string) string {
	if Contains(levels, requested) {
		return requested
	}
	return ""
}
