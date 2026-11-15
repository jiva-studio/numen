// Package process sets what the operating system lets the plugin process use.
package process

import (
	"runtime"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
)

// Process is the running plugin process.
type Process struct{}

// SetMaxProcs sets how many threads run Go code at once.
func (Process) SetMaxProcs(count int) {
	runtime.GOMAXPROCS(count)
}

// LowerNice sets the scheduling priority of every thread of the process to nice.
func (Process) LowerNice(nice int) error {
	return performance.SetNice(nice)
}
