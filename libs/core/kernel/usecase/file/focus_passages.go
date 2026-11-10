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

// maxPassages is the maximum number of passages of one file a focus highlights at once.
const maxPassages = 8

// ErrInvalidPassage is returned by a focus with no path, a passage that starts before the text or is
// empty, or more passages than one file may highlight.
var ErrInvalidPassage = errors.New("invalid passage")

// FocusPassagesInput is the file to show and the passages of it to highlight. The window scrolls to
// the first passage; no passages opens the file at its beginning.
type FocusPassagesInput struct {
	Path     string
	Passages []domain.ByteRange
}

// FocusPassages opens a file of the vault in the window at its passages and highlights them.
type FocusPassages struct {
	readers vaultport.VaultReaders
	view    PlaceFocus
}

// PlaceFocus asks the window to show a place.
type PlaceFocus interface {
	FocusPlace(ctx context.Context, pos domain.Place) error
}

func NewFocusPassages(readers vaultport.VaultReaders, view PlaceFocus) *FocusPassages {
	return &FocusPassages{readers: readers, view: view}
}

// Execute opens the file in the window at its first passage and highlights every passage.
func (u *FocusPassages) Execute(ctx context.Context, v domain.Vault, in FocusPassagesInput) error {
	if err := checkPassages(in); err != nil {
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
	return u.view.FocusPlace(ctx, domain.Place{Path: in.Path, Ranges: in.Passages})
}

func checkPassages(in FocusPassagesInput) error {
	if in.Path == "" {
		return plugin.InvalidFocus(fmt.Errorf("path is empty: %w", ErrInvalidPassage))
	}
	if len(in.Passages) > maxPassages {
		return plugin.InvalidFocus(fmt.Errorf("too many passages: at most %d: %w", maxPassages, ErrInvalidPassage))
	}
	for _, passage := range in.Passages {
		if passage.Start < 0 || passage.Empty() {
			return plugin.InvalidFocus(fmt.Errorf("passage start must be >= 0 and length > 0: %w", ErrInvalidPassage))
		}
	}
	return nil
}
