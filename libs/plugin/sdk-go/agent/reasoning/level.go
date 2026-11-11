package reasoning

import (
	"slices"
	"strings"
)

// Level is one depth of reasoning.
//
//numen:api
type Level struct {
	// Name is the identifier by which a task requests the level.
	Name string `json:"name"`
	// Title is the display name of the level.
	Title string `json:"title"`
}

// Model is one model a program offers.
//
//numen:api
type Model struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	// IsDefault is set where the program names the model it answers with unasked.
	IsDefault bool `json:"isDefault,omitempty"`
	// ReasoningLevels are the depths of reasoning the model can be asked to work at, from the least to
	// the most. Empty where the model offers no choice.
	ReasoningLevels []Level `json:"reasoningLevels,omitempty"`
	// DefaultReasoning is the level the model works at unasked. Empty where it names none.
	DefaultReasoning string `json:"defaultReasoning,omitempty"`
}

func listLevelNames() []string {
	return []string{"minimal", "low", "medium", "high", "xhigh", "max", "ultra"}
}

// NewLevel returns the level for a name, with a display title.
func NewLevel(name string) Level {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "xhigh" {
		return Level{Name: name, Title: "Extra high"}
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
	return slices.Contains(listLevelNames(), strings.ToLower(strings.TrimSpace(name)))
}

func Contains(levels []Level, name string) bool {
	return slices.ContainsFunc(levels, func(one Level) bool { return one.Name == name })
}

// Sort orders the levels from the least reasoning to the most, a name outside the known ones last.
func Sort(levels []Level) {
	order := listLevelNames()
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

// NewModel is a model offered with the levels. The default is empty where it is not one of them.
func NewModel(name, title string, levels []Level, byDefault string) *Model {
	if !Contains(levels, byDefault) {
		byDefault = ""
	}
	return &Model{
		Name: name, Title: title, ReasoningLevels: levels, DefaultReasoning: byDefault,
	}
}

// Normalize returns the requested level if levels contain it, else the default level.
func Normalize(levels []Level, requested string) string {
	if Contains(levels, requested) {
		return requested
	}
	return GetDefault(levels)
}

// Pick returns the requested level if levels contain it, else empty.
func Pick(levels []Level, requested string) string {
	if Contains(levels, requested) {
		return requested
	}
	return ""
}
