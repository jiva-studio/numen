package flashcards

import (
	"errors"

	"golang.org/x/sys/windows"
)

// isLocked says whether a file could not be opened because another program
// holds it. A synchroniser and a backup reader each hold a run file for moments
// at a time, and Windows refuses every other opener while they do.
func isLocked(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_LOCK_VIOLATION)
}
