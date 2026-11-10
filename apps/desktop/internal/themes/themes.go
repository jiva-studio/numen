// Package themes reports the fault of loading the installation's themes.
package themes

import (
	"fmt"
	"io"
)

// Report writes the fault of loading the themes to out. If a folder cannot be created, the themes the
// binary ships are kept, so the fault is all there is to report.
func Report(err error, out io.Writer) {
	if err == nil {
		return
	}
	if _, writeErr := fmt.Fprintf(out, "themes: %v\n", err); writeErr != nil {
		return
	}
}
