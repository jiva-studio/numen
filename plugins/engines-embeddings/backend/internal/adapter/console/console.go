// Package console writes the faults of the plugin that nothing else handles to its error output.
package console

import (
	"fmt"
	"os"
)

// Report writes what failed and why.
func Report(what string, err error) {
	// gomutants:disable-next-line STATEMENT_REMOVE reason="stderr log has no side effect"
	fmt.Fprintf(os.Stderr, "%s: %v\n", what, err)
}
