package derivedfile

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"golang.org/x/sys/windows"
)

func wrapLockedByAnother(err error) error {
	if errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return fmt.Errorf("%w: %w", vault.ErrLockedByAnother, err)
	}
	return err
}
