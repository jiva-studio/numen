package atomicfile

import (
	"errors"
	"syscall"
)

const (
	errorAccessDenied     = syscall.Errno(5)
	errorSharingViolation = syscall.Errno(32)
)

func isFileInUse(err error) bool {
	return errors.Is(err, errorSharingViolation) || errors.Is(err, errorAccessDenied)
}
