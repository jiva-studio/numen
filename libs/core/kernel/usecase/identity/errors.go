package identity

import (
	"errors"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/identity/slip39"
)

func mapSheetError(err error) error {
	if sheetErr, ok := errors.AsType[*identitylib.SheetError](err); ok {
		return &identitydomain.SheetError{Position: sheetErr.Position, IsDuplicate: errors.Is(err, slip39.ErrDuplicateMember), Err: err}
	}
	return err
}

func mapDeviceError(err error) error {
	switch {
	case errors.Is(err, identitylib.ErrSelfRevoke):
		return fmt.Errorf("%w: %w", identitydomain.ErrOwnDevice, err)
	case errors.Is(err, identitylib.ErrUnknownDevice):
		return fmt.Errorf("%w: %w", identitydomain.ErrUnknownDevice, err)
	}
	return err
}

func mapLinkTextError(err error) error {
	if errors.Is(err, identitylib.ErrLinkPayload) {
		return fmt.Errorf("%w: %w", identitydomain.ErrLinkText, err)
	}
	return err
}
