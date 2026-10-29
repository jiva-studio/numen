package atomicfile

import (
	"context"
	"os"
	"time"
)

// waits is how long a rename rests before each further attempt.
var waits = []time.Duration{10 * time.Millisecond, 40 * time.Millisecond, 160 * time.Millisecond}

// Rename moves a file over another, waiting out a handle that another process
// holds on it. A backup reader, a search indexer and a virus scanner each open
// a note for short periods, and the file is locked while they do.
// Both names are inside the root, so neither end of the move can be redirected
// outside it by a link.
func Rename(ctx context.Context, root *os.Root, from, to string) error {
	err := root.Rename(from, to)
	for _, wait := range waits {
		if !isFileInUse(err) {
			return err
		}
		rest := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			rest.Stop()
			return ctx.Err()
		case <-rest.C:
		}
		err = root.Rename(from, to)
	}
	return err
}
