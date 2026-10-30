// Package themes loads the installation's themes into a window.
package themes

import (
	"fmt"
	"io"
)

// Load asks for the installation's themes. A folder that could not be made
// leaves the themes the binary ships, and the fault is written to out.
func Load[S any](get func(say func(string)) (S, error), say func(string), out io.Writer) S {
	loaded, err := get(say)
	if err != nil {
		if _, writeErr := fmt.Fprintf(out, "themes: %v\n", err); writeErr != nil {
			return loaded
		}
	}
	return loaded
}
