// Package hostenv gives the variables a program of the plugin is started with.
package hostenv

import "github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"

// Process gives the variables of the plugin's own process.
type Process struct{}

// GetVariables returns the variables of the process as name=value entries.
func (Process) GetVariables() []string {
	return hostcall.GetChildEnvironment()
}

// Fixed gives the variables it was made from, and none of the process.
type Fixed []string

// GetVariables returns the entries the value was made from.
func (f Fixed) GetVariables() []string {
	return f
}
