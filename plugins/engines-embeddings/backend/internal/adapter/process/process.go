// Package process sets what the operating system lets the plugin process use.
package process

import (
	"runtime"
)

// Process is the running plugin process.
type Process struct{}

// SetMaxProcs sets how many threads run Go code at once.
func (Process) SetMaxProcs(count int) {
	runtime.GOMAXPROCS(count)
}
