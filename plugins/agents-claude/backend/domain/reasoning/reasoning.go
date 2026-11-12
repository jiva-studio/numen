// Package reasoning names the depths of reasoning Claude Code can be asked to work at.
package reasoning

import (
	"slices"
	"strings"
)

// Level is one depth of reasoning.
type Level struct {
	// Name is the identifier a row chooses the level by.
	Name string
	// Title is the display name of the level.
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

// Pick returns the requested level if levels contain it, else empty.
func Pick(levels []Level, requested string) string {
	if slices.ContainsFunc(levels, func(one Level) bool { return one.Name == requested }) {
		return requested
	}
	return ""
}
