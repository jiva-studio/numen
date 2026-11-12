// Package themes reports the fault of loading the installation's themes.
package themes

import (
	"fmt"
	"io"
)

// Report writes the fault of loading the themes to out, and nothing when there is none. The window
// keeps the themes the binary ships when a folder cannot be created.
func Report(err error, out io.Writer) {
	if err == nil {
		return
	}
	if _, writeErr := fmt.Fprintf(out, "themes: %v\n", err); writeErr != nil {
		return
	}
}
