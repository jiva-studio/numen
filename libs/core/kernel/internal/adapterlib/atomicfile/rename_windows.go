package atomicfile

import (
	"errors"

	"golang.org/x/sys/windows"
)

func isFileInUse(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_ACCESS_DENIED)
}
