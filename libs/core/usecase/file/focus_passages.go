package file

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// mostPassages is how many passages of one file a focus lights at once.
const mostPassages = 8

// ErrInvalidPassage is what a focus gets for no path, a passage that starts before the text or is
// empty, or more passages than one file may light.
var ErrInvalidPassage = errors.New("the passages cannot be shown")

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
func (u *FocusPassages) Execute(ctx context.Context, v domain.Vault, in FocusPassagesInput) (string, error) {
	if err := checkPassages(in); err != nil {
		return "", err
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return "", err
	}
	stat, err := reader.StatFile(ctx, in.Path)
	if vaultport.IsNoNote(err) {
		return "", fmt.Errorf("this vault holds nothing at %s: %w", in.Path, vaultport.ErrNotExist)
	}
	if err != nil {
		return "", err
	}
	if err := u.view.FocusPlace(ctx, domain.Place{Path: in.Path, Ranges: in.Passages}); err != nil {
		return "", err
	}
	return getFocusSummary(stat.Kind == domain.KindNote, len(in.Passages)), nil
}

// getFocusSummary returns a sentence describing what the person now sees.
func getFocusSummary(isNote bool, passages int) string {
	switch {
	case isNote:
		return "the note is in front of them, and a note is shown whole"
	case passages == 0:
		return "the document is open at its beginning"
	}
	return "the document is open at that passage, with the words of it lit"
}

// checkPassages returns an error for an input with no file, a passage that starts before the text
// or is empty, or more passages than one file may light.
func checkPassages(in FocusPassagesInput) error {
	if in.Path == "" {
		return fmt.Errorf("name the document to show: %w", ErrInvalidPassage)
	}
	if len(in.Passages) > mostPassages {
		return fmt.Errorf("light at most %d places of one document: %w", mostPassages, ErrInvalidPassage)
	}
	for _, passage := range in.Passages {
		if passage.Start < 0 || passage.Empty() {
			return fmt.Errorf(
				"a passage begins at or after the start of the text, and is longer than nothing: %w", ErrInvalidPassage)
		}
		// The window receives the ranges as 32-bit offsets.
		if passage.End > math.MaxInt32 {
			return fmt.Errorf("a passage ends at most %d bytes into the text: %w", math.MaxInt32, ErrInvalidPassage)
		}
	}
	return nil
}
