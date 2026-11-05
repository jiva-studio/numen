package file

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// ErrInvalidPassage is what a focus gets for no path, a passage that starts before the text or is
// empty, or more passages than one file may light.
var ErrInvalidPassage = errors.New("the passages cannot be shown")

var errFocusNotImplemented = errors.New("focus passages is not implemented")

// FocusPassagesInput is the file to show and the passages of it to light. The first passage is the
// one the window goes to; none opens the file at its beginning.
type FocusPassagesInput struct {
	Path     string
	Passages []domain.ByteRange
}

// FocusPassages opens a file of the vault in front of the person at its passages, with the words of
// them lit.
type FocusPassages struct {
	readers vaultport.VaultReaders
	view    windowport.Window
}

func NewFocusPassages(readers vaultport.VaultReaders, view windowport.Window) *FocusPassages {
	return &FocusPassages{readers: readers, view: view}
}

// Execute puts the passages in front of the person and returns a sentence saying what they now see.
func (u *FocusPassages) Execute(_ context.Context, _ domain.Vault, _ FocusPassagesInput) (string, error) {
	return "", errFocusNotImplemented
}
