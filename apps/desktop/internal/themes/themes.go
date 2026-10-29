// Package themes loads the installation's themes into a window.
package themes

import (
	"fmt"
	"io"
)

// Load fetches the installation's themes. If a folder cannot be created, the
// themes the binary ships are kept and the fault is written to out.
func Load[S any](get func(say func(string)) (S, error), say func(string), out io.Writer) S {
	loaded, err := get(say)
	if err != nil {
		fmt.Fprintf(out, "themes: %v\n", err)
	}
	return loaded
}
