package host

import (
	"fmt"
	"os"
)

// reportError writes an error that has no caller to return to onto standard error, which the host
// collects as the plugin's log.
func reportError(err error) {
	if _, writeErr := fmt.Fprintln(os.Stderr, "host:", err); writeErr != nil {
		return
	}
}

// closeStream closes stream and reports a failure to close.
func closeStream(stream interface{ Close() error }) {
	if err := stream.Close(); err != nil {
		reportError(fmt.Errorf("close stream: %w", err))
	}
}
