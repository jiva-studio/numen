//go:build !windows

package flashcards

// isLocked says whether a file could not be opened because another program
// holds it. Here a file is read whatever else has it open.
func isLocked(error) bool { return false }
