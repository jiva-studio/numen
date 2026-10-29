package derivedfile

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// wrapLockedByAnother wraps vaultport.ErrLockedByAnother around err when another
// process holds the file, and returns err itself otherwise.
//
// Windows denies every other opener while a program has a file open, and reports
// this with two error codes.
func wrapLockedByAnother(err error) error {
	if errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return fmt.Errorf("%w: %w", vaultport.ErrLockedByAnother, err)
	}
	return err
}
