package editor

import (
	"fmt"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
)

// writingDiagnostics is the entry in the list of what is being done for a
// diagnostic line that could not be written.
const writingDiagnostics = "writing diagnostics"

// closingResources is the entry for a file or response body that could not be closed.
const closingResources = "closing a file"

func writeDiagnostic(report func(vaultreading.Task), out io.Writer, format string, a ...any) {
	if _, err := fmt.Fprintf(out, format, a...); err != nil {
		report(vaultreading.Task{ID: writingDiagnostics, Title: "Writing diagnostics", Error: err.Error()})
	}
}

func closeAndReport(report func(vaultreading.Task), c io.Closer) {
	if err := c.Close(); err != nil {
		report(vaultreading.Task{ID: closingResources, Title: "Closing a file", Error: err.Error()})
	}
}
