package text_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
)

var atlas = domain.Fingerprint{Path: "books/atlas.epub", Kind: domain.KindBook}

// paged is a reader of books that names a printed page.
type paged struct{ testsupport.Documents }

func (paged) ReadBook(context.Context, []byte) (domain.BookText, error) {
	return domain.BookText{
		Text:  "alpha beta\ngamma",
		Parts: []domain.PartStart{{Title: "One", Offset: 0}, {Title: "Two", Offset: 11}},
		Pages: []domain.BookPage{{Label: "iv", Offset: 0}, {Label: "", Offset: 5}, {Label: "v", Offset: 11}},
	}, nil
}

// failing is a reader of books that cannot read the book it is given.
type failing struct {
	testsupport.Documents
	err error
}

func (f failing) ReadBook(context.Context, []byte) (domain.BookText, error) {
	return domain.BookText{}, f.err
}

func TestABookIsReadByTheReaderOfBooksGiven(t *testing.T) {
	doc, err := text.Read(t.Context(), paged{}, atlas, []byte("PK"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Text != "alpha beta\ngamma" {
		t.Errorf("text = %q, want the text the reader answered", doc.Text)
	}
	if len(doc.Parts) != 2 || doc.Parts[1].Title != "Two" || doc.Parts[1].Offset != 11 {
		t.Errorf("parts = %+v, want the two the book names", doc.Parts)
	}
	if got := doc.Locate(12); got != "Two, v" {
		t.Errorf("locate(12) = %q, want the part and the printed page the offset falls in", got)
	}
}

func TestABookWithoutAReaderOfBooksIsUnreadable(t *testing.T) {
	doc, err := text.Read(t.Context(), nil, atlas, []byte("PK"))
	if doc != nil || !errors.Is(err, text.ErrUnreadable) {
		t.Fatalf("answered %+v, %v, want nothing and %v", doc, err, text.ErrUnreadable)
	}
}

func TestABookTheReaderRefusesIsUnreadable(t *testing.T) {
	for _, readErr := range []error{sourceport.ErrNotADocument, sourceport.ErrEncrypted, errors.New("the plugin is gone")} {
		doc, err := text.Read(t.Context(), failing{err: readErr}, atlas, []byte("PK"))
		if doc != nil || !errors.Is(err, text.ErrUnreadable) {
			t.Errorf("%v: answered %+v, %v, want nothing and %v", readErr, doc, err, text.ErrUnreadable)
		}
	}
}

func TestABookReadAfterTheCallerGaveUpAnswersWhyItStopped(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := text.Read(ctx, failing{err: context.Canceled}, atlas, []byte("PK"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the cancellation", err)
	}
}
