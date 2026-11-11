// Package diagnostics writes diagnostics of the plugin process to its standard error.
package diagnostics

import (
	"fmt"
	"os"
)

// ReportSyncError writes the failure of a sync of the host tools.
func ReportSyncError(err error) {
	fmt.Fprintf(os.Stderr, "sync the host tools: %v\n", err)
}
