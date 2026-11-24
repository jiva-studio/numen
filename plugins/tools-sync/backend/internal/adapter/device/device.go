// Package device names the computer the plugin runs on, from its host name.
package device

import "os"

// Host names the computer by its host name.
type Host struct{}

// GetName returns the host name, or an empty text when the system gives none.
func (Host) GetName() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return name
}
