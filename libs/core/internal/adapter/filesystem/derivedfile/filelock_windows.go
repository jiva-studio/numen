package derivedfile

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// somebody else holding it, and err itself otherwise.
//
// Windows refuses every other opener while a program has a file open, and says
// so in two codes.
func wrapLockedByAnother(err error) error {
	if errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return fmt.Errorf("%w: %w", vaultport.ErrLockedByAnother, err)
	}
	return err
}
