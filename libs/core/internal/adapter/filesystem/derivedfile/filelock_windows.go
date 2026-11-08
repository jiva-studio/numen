package derivedfile

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

func wrapLockedByAnother(err error) error {
	if errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return fmt.Errorf("%w: %w", vaultport.ErrLockedByAnother, err)
	}
	return err
}
