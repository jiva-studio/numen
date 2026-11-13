package plugin

import (
	"slices"
	"time"
)

// Process is the process of a plugin that runs. LastUseTime is when a lease was last taken or
// returned. IdleStopTime is from when the core stops the process for idleness, and zero while it
// holds a lease or runs inside the core.
type Process struct {
	PluginID     string
	StartTime    time.Time
	LastUseTime  time.Time
	LeaseCount   int
	IsInProcess  bool
	IdleStopTime time.Time
	StartedBy    Holder
	Holders      []Holder
}

// ProcessStatus is a process that runs, with the name of its plugin and the worst health item of the
// plugin. Health is HealthOK where the plugin has no item.
type ProcessStatus struct {
	Process
	PluginName string
	Health     HealthSeverity
	HealthText string
}

// GetWorstHealth returns the item of the plugin with the highest severity, the first reported among
// equals, and false where the plugin has none.
func GetWorstHealth(items []HealthItem, pluginID string) (HealthItem, bool) {
	var worst HealthItem
	isFound := false
	for _, item := range items {
		if item.PluginID != pluginID || item.Severity <= HealthOK {
			continue
		}
		if !isFound || item.Severity > worst.Severity {
			worst, isFound = item, true
		}
	}
	return worst, isFound
}

// ProcessList is the processes that run, in the order of the plugin ids, as they were at Time.
type ProcessList struct {
	Time      time.Time
	Processes []ProcessStatus
}

// IsSameStatus reports whether two statuses say the same.
func IsSameStatus(a, b ProcessStatus) bool {
	return a.PluginID == b.PluginID && a.StartTime == b.StartTime && a.LastUseTime == b.LastUseTime &&
		a.LeaseCount == b.LeaseCount && a.IsInProcess == b.IsInProcess && a.IdleStopTime == b.IdleStopTime &&
		a.StartedBy == b.StartedBy && slices.Equal(a.Holders, b.Holders) &&
		a.PluginName == b.PluginName && a.Health == b.Health && a.HealthText == b.HealthText
}
