package windowsession

import (
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
)

// writingDiagnostics is the entry in the list of what is being done for a diagnostic line that could
// not be written.
const writingDiagnostics = "writing diagnostics"

// Diagnostics writes the lines the window says about itself. A line that cannot be written becomes a
// task, so it is not lost.
type Diagnostics struct {
	out   io.Writer
	tasks windowport.Tasks
}

// NewDiagnostics makes the diagnostics that write to out and report a failed write to tasks. A nil out
// discards the lines.
func NewDiagnostics(out io.Writer, tasks windowport.Tasks) *Diagnostics {
	if out == nil {
		out = io.Discard
	}
	return &Diagnostics{out: out, tasks: tasks}
}

func (d *Diagnostics) say(format string, a ...any) {
	if _, err := fmt.Fprintf(d.out, format, a...); err != nil {
		d.tasks.AddTask(work.Task{ID: writingDiagnostics, Title: "Writing diagnostics", Error: err.Error()})
	}
}
