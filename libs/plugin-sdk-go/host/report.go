package host

import (
	"fmt"
	"os"
)

// reportError writes err to standard error. It serves code paths that have no caller to return err to.
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
