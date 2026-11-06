package file

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// mostPassages is how many passages of one file a focus highlights at once.
const mostPassages = 8

// ErrInvalidPassage is what a focus gets for no path, a passage that starts before the text or is
// empty, or more passages than one file may highlight.
var ErrInvalidPassage = errors.New("the passages cannot be shown")

// FocusPassagesInput is the file to show and the passages of it to highlight. The first passage is the
// one the window goes to; none opens the file at its beginning.
type FocusPassagesInput struct {
	Path     string
	Passages []domain.ByteRange
}

// FocusPassages opens a file of the vault in the window at its passages and highlights them.
type FocusPassages struct {
	readers vaultport.VaultReaders
	view    windowport.Window
}

func NewFocusPassages(readers vaultport.VaultReaders, view windowport.Window) *FocusPassages {
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
	if vaultport.IsNoNote(err) {
		return fmt.Errorf("this vault holds nothing at %s: %w", in.Path, vaultport.ErrNotExist)
	}
	if err != nil {
		return err
	}
	return u.view.FocusPlace(ctx, domain.Place{Path: in.Path, Ranges: in.Passages})
}

// checkPassages returns an error for an input with no file, a passage that starts before the text
// or is empty, or more passages than one file may highlight.
func checkPassages(in FocusPassagesInput) error {
	if in.Path == "" {
		return fmt.Errorf("name the document to show: %w", ErrInvalidPassage)
	}
	if len(in.Passages) > mostPassages {
		return fmt.Errorf("highlight at most %d places of one document: %w", mostPassages, ErrInvalidPassage)
	}
	for _, passage := range in.Passages {
		if passage.Start < 0 || passage.Empty() {
			return fmt.Errorf(
				"a passage begins at or after the start of the text, and is longer than nothing: %w", ErrInvalidPassage)
		}
	}
	return nil
}
