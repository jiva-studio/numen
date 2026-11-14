// Package reasoning names the depths of reasoning an Antigravity model can work at, folds the models
// the program lists once per depth into one model with levels, and converts a chosen level back into the
// model the program is started with.
package reasoning

import (
	"slices"
	"strings"
)

// Level is one depth of reasoning. A row chooses it by Name; Title is what a person reads.
type Level struct {
	Name  string
	Title string
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

// NewLevels returns the levels of the names, in the order given.
func NewLevels(names ...string) []Level {
	levels := make([]Level, 0, len(names))
	for _, name := range names {
		levels = append(levels, NewLevel(name))
	}
	return levels
}

// IsKnown reports whether the name is one of the level names.
func IsKnown(name string) bool {
	return slices.Contains(listLevelNames(), strings.ToLower(strings.TrimSpace(name)))
}

// Contains reports whether the levels hold one of the name.
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

// GetDefault returns the level a model works at where a row names none: "medium" where the levels hold
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

// Normalize returns the requested level if levels contain it, else the default level.
func Normalize(levels []Level, requested string) string {
	if Contains(levels, requested) {
		return requested
	}
	return GetDefault(levels)
}
