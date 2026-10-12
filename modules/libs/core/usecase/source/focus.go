package source

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Passage is where in a document to look. A Length of zero opens the document
// at its beginning, and Also is the other passages of the document to light.
type Passage struct {
	Path   string
	Start  int
	Length int
	Also   []domain.ByteSpan
}

// FocusPassage opens one of the vault's documents in front of the person at a
// passage, with the words of it lit.
type FocusPassage struct {
	readers port.VaultReaders
	view    port.Window
}

func NewFocusPassage(readers port.VaultReaders, view port.Window) *FocusPassage {
	return &FocusPassage{readers: readers, view: view}
}

// Execute puts the passage in front of the person and returns the kind of the
// file it opened. A file the vault does not hold, a passage that starts before
// the text and more passages than a document may highlight are refused.
func (u *FocusPassage) Execute(ctx context.Context, v domain.Vault, in Passage) (domain.SourceKind, error) {
	if err := checkPassage(in); err != nil {
		return "", err
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return "", err
	}
	ref, err := reader.GetFingerprint(ctx, in.Path)
	if port.IsNoNote(err) {
		return "", fmt.Errorf("this vault holds nothing at %s", in.Path)
	}
	if err != nil {
		return "", err
	}
	at, err := newPlace(in)
	if err != nil {
		return "", err
	}
	if err := u.view.FocusPlace(ctx, at); err != nil {
		return "", err
	}
	return ref.Kind, nil
}

// checkPassage returns an error for an input with no document, a passage that
// starts before the text, or more passages than a document may highlight.
func checkPassage(in Passage) error {
	if in.Path == "" {
		return errors.New("name the document to show")
	}
	if in.Start < 0 || in.Length < 0 {
		return errors.New(
			"a passage begins at or after the start of the text, and its length is zero or more")
	}
	if 1+len(in.Also) > domain.MostHighlights {
		return fmt.Errorf("light at most %d places of one document", domain.MostHighlights)
	}
	return nil
}

// newPlace returns the place for the document and passages named in the input.
func newPlace(in Passage) (domain.Place, error) {
	at := domain.Place{Path: in.Path}
	if in.Length > 0 {
		at.Spans = append(at.Spans, domain.ByteSpan{From: in.Start, To: in.Start + in.Length})
	}
	for _, one := range in.Also {
		if one.From < 0 || one.To <= one.From {
			return domain.Place{}, errors.New(
				"a passage begins at or after the start of the text, and is longer than nothing")
		}
		at.Spans = append(at.Spans, one)
	}
	return at, nil
}
