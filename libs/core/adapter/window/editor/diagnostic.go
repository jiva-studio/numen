package editor

import (
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/core/usecase/run"
)

// writingDiagnostics is the entry in the list of what is being done for a
// diagnostic line that could not be written.
const writingDiagnostics = "writing diagnostics"

// closingResources is the entry for a file or response body that could not be closed.
const closingResources = "closing a file"

// relayingResponse is the entry for a response that could not be copied to its client.
const relayingResponse = "relaying a response"

// writeDiagnostic writes a formatted line to out. A failed write is reported
// through report as a failed task.
func writeDiagnostic(report func(run.Task), out io.Writer, format string, a ...any) {
	if _, err := fmt.Fprintf(out, format, a...); err != nil {
		report(run.Task{ID: writingDiagnostics, Title: "Writing diagnostics", Error: err.Error()})
	}
}

// closeAndReport closes c. A failed close is reported through report as a failed task.
func closeAndReport(report func(run.Task), c io.Closer) {
	if err := c.Close(); err != nil {
		report(run.Task{ID: closingResources, Title: "Closing a file", Error: err.Error()})
	}
}
