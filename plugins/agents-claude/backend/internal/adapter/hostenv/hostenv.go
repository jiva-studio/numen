// Package hostenv gives the variables a program of the plugin is started with.
package hostenv

import "os"

// Process gives the variables of the plugin's own process.
type Process struct{}

// GetVariables returns the variables of the process as name=value entries.
func (Process) GetVariables() []string {
	return os.Environ()
}

// Fixed gives the variables it was made from, and none of the process.
type Fixed []string

// GetVariables returns the entries the value was made from.
func (f Fixed) GetVariables() []string {
	return f
}
