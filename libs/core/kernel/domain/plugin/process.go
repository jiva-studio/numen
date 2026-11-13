package plugin

import "time"

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
