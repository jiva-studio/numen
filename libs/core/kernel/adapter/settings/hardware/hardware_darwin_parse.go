//go:build darwin

package hardware

import "strings"

// perfLevel is one sysctl performance level of an Apple chip.
type perfLevel struct {
	Name         string
	PhysicalCPUs int
	LogicalCPUs  int
}

func getPerfLevelRank(name string) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "super":
		return 3
	case "performance":
		return 2
	case "efficiency":
		return 1
	default:
		return 0
	}
}

func pickPerformanceLevel(levels []perfLevel) (perfLevel, bool) {
	best := -1
	for i, level := range levels {
		if level.PhysicalCPUs < 1 {
			continue
		}
		if best < 0 || getPerfLevelRank(level.Name) > getPerfLevelRank(levels[best].Name) {
			best = i
		}
	}
	if best < 0 {
		return perfLevel{}, false
	}
	return levels[best], true
}
