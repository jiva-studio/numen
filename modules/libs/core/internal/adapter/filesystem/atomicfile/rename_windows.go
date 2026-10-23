package atomicfile

import (
	"errors"

	"golang.org/x/sys/windows"
)

// isFileInUse says whether a rename failed because another program holds the file
// open. Windows refuses the move for as long as the handle stands.
func isFileInUse(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_ACCESS_DENIED)
}
