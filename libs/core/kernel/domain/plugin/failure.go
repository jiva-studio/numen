package plugin

import "time"

// Failure is a plugin the host suspends after it failed to start, and why.
type Failure struct {
	PluginID string
	Name     string
	Reason   string
	Detail   string
	At       time.Time
}
