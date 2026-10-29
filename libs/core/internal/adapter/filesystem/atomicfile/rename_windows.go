package atomicfile

import (
	"errors"

	"golang.org/x/sys/windows"
)

// isFileInUse reports whether a rename failed because another program has the
// file open. Windows rejects the move while the handle is open.
func isFileInUse(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_ACCESS_DENIED)
}
