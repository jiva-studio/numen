package filesystem

import (
	"context"
	"os"
	"time"
)

// waits is how long a rename rests before each further attempt.
var waits = []time.Duration{10 * time.Millisecond, 40 * time.Millisecond, 160 * time.Millisecond}

// rename moves a file over another, waiting out a handle somebody else holds on
// it. A backup reader, a search indexer and a virus scanner each open a note for
// moments at a time, and the file is theirs while they do.
// Both names are the root's own, so neither end of the move can be carried out
// of it by a link put in the way.
func rename(ctx context.Context, root *os.Root, from, to string) error {
	err := root.Rename(from, to)
	for _, wait := range waits {
		if !transient(err) {
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
