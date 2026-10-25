package source

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// FocusPassage opens one of the vault's documents in front of the person at a
// passage, with the words of it lit.
type FocusPassage struct {
	readers vaultport.VaultReaders
	view    windowport.Window
}

func NewFocusPassage(readers vaultport.VaultReaders, view windowport.Window) *FocusPassage {
	return &FocusPassage{readers: readers, view: view}
}

// Execute puts the passage in front of the person and returns a sentence saying what they now see.
func (u *FocusPassage) Execute(ctx context.Context, v domain.Vault, in Passage) (string, error) {
	if err := checkPassage(in); err != nil {
		return "", err
	}
	if u.readers == nil {
		return "", errors.New("this vault's files are not open")
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return "", err
	}
	ref, err := reader.GetFingerprint(ctx, in.Path)
	if vaultport.IsNoNote(err) {
		return "", fmt.Errorf("this vault holds nothing at %s", in.Path)
	}
	if err != nil {
		return "", err
	}
	pos, err := newPlace(in)
	if err != nil {
		return "", err
	}
	if err := u.view.FocusPlace(ctx, pos); err != nil {
		return "", err
	}
	return getFocusSummary(ref.Kind, in.Length), nil
}

// getFocusSummary returns a sentence describing what the person now sees.
func getFocusSummary(kind domain.SourceKind, length int) string {
	switch {
	case kind == domain.KindNote:
		return "the note is in front of them, and a note is shown whole"
	case length == 0:
		return "the document is open at its beginning"
	}
	return "the document is open at that passage, with the words of it lit"
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
	pos := domain.Place{Path: in.Path}
	if in.Length > 0 {
		pos.Spans = append(pos.Spans, domain.ByteSpan{From: in.Start, To: in.Start + in.Length})
	}
	for _, one := range in.Also {
		if one.From < 0 || one.To <= one.From {
			return domain.Place{}, errors.New(
				"a passage begins at or after the start of the text, and is longer than nothing")
		}
		pos.Spans = append(pos.Spans, one)
	}
	return pos, nil
}
