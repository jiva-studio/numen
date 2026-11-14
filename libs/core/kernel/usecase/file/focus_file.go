package file

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// maxRanges is the maximum number of byte ranges of one file a focus highlights at once.
const maxRanges = 8

// ErrInvalidRange is returned by a focus with no path, a range that starts before the text or is
// empty, or more ranges than one file may highlight.
var ErrInvalidRange = errors.New("invalid passage")

// FocusFileInput is the file to show and the byte ranges of it to highlight. The window scrolls to
// the first range; no ranges opens the file at its beginning.
type FocusFileInput struct {
	Path   string
	Ranges []domain.ByteRange
}

// FocusFile opens a file of the vault in the window at its byte ranges and highlights them.
type FocusFile struct {
	readers vaultport.VaultReaders
	view    PlaceFocus
}

// PlaceFocus asks the window to show a place.
type PlaceFocus interface {
	FocusPlace(ctx context.Context, pos domain.Place) error
}

func NewFocusFile(readers vaultport.VaultReaders, view PlaceFocus) *FocusFile {
	return &FocusFile{readers: readers, view: view}
}

// Execute opens the file in the window at its first range and highlights every range.
func (u *FocusFile) Execute(ctx context.Context, v domain.Vault, in FocusFileInput) error {
	if err := checkRanges(in); err != nil {
		return err
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return err
	}
	_, err = reader.StatFile(ctx, in.Path)
	if edit.IsNoNote(err) {
		return fmt.Errorf("no file at %s: %w", in.Path, edit.ErrNotExist)
	}
	if err != nil {
		return err
	}
	return u.view.FocusPlace(ctx, domain.Place{Path: in.Path, Ranges: in.Ranges})
}

func checkRanges(in FocusFileInput) error {
	if in.Path == "" {
		return plugin.InvalidFocus(fmt.Errorf("path is empty: %w", ErrInvalidRange))
	}
	if len(in.Ranges) > maxRanges {
		return plugin.InvalidFocus(fmt.Errorf("too many passages: at most %d: %w", maxRanges, ErrInvalidRange))
	}
	for _, r := range in.Ranges {
		if r.Start < 0 || r.Empty() {
			return plugin.InvalidFocus(fmt.Errorf("passage start must be >= 0 and length > 0: %w", ErrInvalidRange))
		}
	}
	return nil
}
