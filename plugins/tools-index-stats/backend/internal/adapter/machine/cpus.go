// Package machine reads facts of the computer the plugin runs on.
package machine

import "runtime"

// GetCPUCount is the count of logical processors the process can use.
func GetCPUCount() int { return runtime.NumCPU() }
