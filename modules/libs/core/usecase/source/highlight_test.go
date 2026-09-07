package source

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/correction"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// layered reads a document with the library and answers where its words sit
// with what the test put in.
type layered struct {
	port.TextExtractor
	where func(raw []byte, pages []int) ([]highlight.Box, error)
}

func (l layered) Highlights(_ context.Context, raw []byte, _ []int, pages []int) ([]highlight.Box, error) {
	return l.where(raw, pages)
}

// answering is the use case with the test's own answer for where words sit.
func answering(u Highlight, where func(raw []byte, pages []int) ([]highlight.Box, error)) Highlight {
	u.Documents = layered{TextExtractor: documents{}, where: where}
	return u
}

// placing is a Highlight over one vault holding one document, and the document read.
//
// The document is pages of a few words each, so a test can name a word and say
// which page it is printed on.
func placing(t *testing.T, pages [][]string) (Highlight, *store, *shelf, document, *library) {
	t.Helper()
	raw := printedAs(pages)
	book := documentOf(raw)
	shelved := newLibrary()
	shelved.hold(documentPath, domain.KindBook, raw, 1)
	index := newStore()
	store := newShelf()
	return Highlight{
		Readers:   vaults{first.ID: shelved},
		Sources:   index,
		Derived:   shelves{store},
		Documents: documents{},
	}, index, store, book, shelved
}

// holds records a source the way the index holds one: the file as it stood
// when it was read, and the producer of the text its chunks are places in.
func holds(t *testing.T, index *store, shelved *library, from, hash string) {
	t.Helper()
	ref, err := shelved.Stat(t.Context(), documentPath)
	if err != nil {
		t.Fatal(err)
	}
	index.put(first.ID, domain.Source{Fingerprint: ref, Hash: hash, Producer: from})
}

// run is where a word of the document is: its offset in the text, and how long
// it is.
func run(t *testing.T, book document, word string) (start, length int) {
	t.Helper()
	at := strings.Index(book.Text, word)
	if at < 0 {
		t.Fatalf("the document does not say %q", word)
	}
	return at, len(word)
}

// litOn is where one run of a source's text sits, asked about on its own. A
// source with no reading and no layer is lit nowhere.
func litOn(t *testing.T, u Highlight, path string, start, length int) []highlight.Page {
	t.Helper()
	found, err := u.Execute(t.Context(), first, path, []highlight.Stretch{{Start: start, Length: length}})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		return nil
	}
	return found[0]
}

// A run of a recognised source's text is lit from what the model wrote down.
// The document's own layer is not asked: the offsets are places in the text the
// model produced, and the layer's words are elsewhere in the book.
func TestARecognisedSourceIsLitFromWhatWasReadInIt(t *testing.T) {
	u, index, store, _, shelved := placing(t, tiny)
	holds(t, index, shelved, "ocr", "abc123")
	u = answering(u, func([]byte, []int) ([]highlight.Box, error) {
		t.Error("the document's own layer was read for a source standing on a reading")
		return nil, nil
	})

	// Two words on one page and a third on the next, as a model reading the
	// pages wrote them down.
	written := []highlight.Box{
		testsupport.Box(4, 0, 5, highlight.Rect{MinX: 0.1, MinY: 0.2, MaxX: 0.2, MaxY: 0.23}),
		testsupport.Box(4, 6, 4, highlight.Rect{MinX: 0.21, MinY: 0.2, MaxX: 0.3, MaxY: 0.23}),
		testsupport.Box(5, 11, 5, highlight.Rect{MinX: 0.1, MinY: 0.5, MaxX: 0.2, MaxY: 0.53}),
	}
	if err := store.Write(t.Context(), text.Boxes("ocr", "abc123"), highlight.Pack(written)); err != nil {
		t.Fatal(err)
	}

	found := litOn(t, u, documentPath, 0, 10)
	if len(found) != 1 || found[0].Index != 4 || len(found[0].Rects) != 2 {
		t.Fatalf("the run was lit at %+v", found)
	}
	want := highlight.Rect{MinX: 0.1, MinY: 0.2, MaxX: 0.2, MaxY: 0.23}
	if found[0].Rects[0] != want {
		t.Errorf("the first word is at %+v, want %+v", found[0].Rects[0], want)
	}
}

// The same source before anything read it is lit from the document's own text
// layer, over the page the word is printed on.
func TestASourceWithNoReadingIsLitFromItsOwnLayer(t *testing.T) {
	u, index, _, book, shelved := placing(t, tiny)
	holds(t, index, shelved, "", "abc123")

	start, length := run(t, book, "gamma")
	found := litOn(t, u, documentPath, start, length)
	if len(found) != 1 || found[0].Index != 0 || len(found[0].Rects) != 1 {
		t.Fatalf("%q was lit at %+v", "gamma", found)
	}

	// The rectangle is the one the document puts that word in.
	var want highlight.Rect
	for _, box := range book.boxes([]int{0}) {
		if box.Start == start {
			want = highlight.Rect{MinX: box.MinX, MinY: box.MinY, MaxX: box.MaxX, MaxY: box.MaxY}
		}
	}
	if found[0].Rects[0] != want {
		t.Errorf("%q is at %+v, want the %+v the document places it at",
			"gamma", found[0].Rects[0], want)
	}
}

// A run that crosses a page comes back as two pages, each with the words of it
// that are printed there.
func TestARunCrossingAPageIsOnBothOfThem(t *testing.T) {
	u, index, _, book, shelved := placing(t, tiny)
	holds(t, index, shelved, "", "abc123")

	from, _ := run(t, book, "gamma")
	to, length := run(t, book, "Delta")
	found := litOn(t, u, documentPath, from, to+length-from)
	if len(found) != 2 || found[0].Index != 0 || found[1].Index != 1 {
		t.Fatalf("a run across a page was lit at %+v", found)
	}
	for _, page := range found {
		if len(page.Rects) != 1 {
			t.Errorf("page %d lights %d words, want the one printed on it",
				page.Index, len(page.Rects))
		}
	}
}

// A word on the last page is lit on the last page. Where a run falls is worked
// out from where each page's text begins, so the rectangles land where the word
// is printed.
func TestAWordIsLitOnThePageItIsPrintedOn(t *testing.T) {
	u, index, _, book, shelved := placing(t, outline)
	holds(t, index, shelved, "", "abc123")

	start, length := run(t, book, "Afterword")
	found := litOn(t, u, documentPath, start, length)
	if len(found) != 1 || found[0].Index != 3 {
		t.Fatalf("%q is printed on page 3 and was lit at %+v", "Afterword", found)
	}
}

// One page of a document is lit, and the rest of it is not. A question about
// a paragraph is not a reason to read a book.
func TestOnlyThePagesARunFallsOnAreLit(t *testing.T) {
	u, index, _, book, shelved := placing(t, outline)
	holds(t, index, shelved, "", "abc123")

	var asked []int
	u = answering(u, func(_ []byte, pages []int) ([]highlight.Box, error) {
		asked = pages
		return book.boxes(pages), nil
	})

	start, length := run(t, book, "closer")
	found := litOn(t, u, documentPath, start, length)
	if !slices.Equal(asked, []int{2}) {
		t.Errorf("a word on page 2 of %d asked for pages %v", len(book.Pages), asked)
	}
	if len(found) != 1 || found[0].Index != 2 {
		t.Errorf("%q was lit at %+v", "closer", found)
	}
}

// Several places are asked about at once and come back in the order they were
// asked about, so a caller knows which answer is which. The pages they fall on
// are read once, however many of the places stand on one page.
func TestSeveralPlacesAreAskedAboutAtOnce(t *testing.T) {
	u, index, _, book, shelved := placing(t, outline)
	holds(t, index, shelved, "", "abc123")

	var asked [][]int
	u = answering(u, func(_ []byte, pages []int) ([]highlight.Box, error) {
		asked = append(asked, pages)
		return book.boxes(pages), nil
	})

	after, afterLength := run(t, book, "Afterword")
	closer, closerLength := run(t, book, "closer")
	found, err := u.Execute(t.Context(), first, documentPath, []highlight.Stretch{
		{Start: after, Length: afterLength},
		{Start: closer, Length: closerLength},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Fatalf("two places were asked about and %d came back: %+v", len(found), found)
	}
	if len(found[0]) != 1 || found[0][0].Index != 3 {
		t.Errorf("%q is printed on page 3 and was lit at %+v", "Afterword", found[0])
	}
	if len(found[1]) != 1 || found[1][0].Index != 2 {
		t.Errorf("%q is printed on page 2 and was lit at %+v", "closer", found[1])
	}
	if len(asked) != 1 || !slices.Equal(asked[0], []int{2, 3}) {
		t.Errorf("the layer was asked for %v, want both pages once", asked)
	}
}

// A path the vault does not hold is refused, and so is one leaving it. Nothing
// is read and nothing is lit.
func TestAPathTheVaultDoesNotHoldIsRefused(t *testing.T) {
	u, index, _, _, shelved := placing(t, tiny)
	holds(t, index, shelved, "", "abc123")

	for _, path := range []string{"library/nothing.pdf", "../outside.pdf"} {
		t.Run(path, func(t *testing.T) {
			found, err := u.Execute(t.Context(), first, path, []highlight.Stretch{{Start: 0, Length: 5}})
			if err == nil {
				t.Errorf("%s was answered with %+v", path, found)
			}
		})
	}
}

// A source the index does not hold is lit nowhere: which producer made its
// text is what says where its offsets are, and nothing has said.
func TestASourceTheIndexDoesNotHoldIsLitNowhere(t *testing.T) {
	u, _, _, book, _ := placing(t, tiny)
	u = answering(u, func([]byte, []int) ([]highlight.Box, error) {
		t.Error("a source the index does not hold was read")
		return nil, nil
	})

	start, length := run(t, book, "gamma")
	found := litOn(t, u, documentPath, start, length)
	if len(found) != 0 {
		t.Errorf("a source the index does not hold was lit at %+v", found)
	}
}

// A run past the end of the text is lit nowhere, and a reading that is not
// there says nothing.
func TestNothingIsLitWhereThereIsNothingToLight(t *testing.T) {
	t.Run("past the end", func(t *testing.T) {
		u, index, _, book, shelved := placing(t, tiny)
		holds(t, index, shelved, "", "abc123")

		if found := litOn(t, u, documentPath, len(book.Text)+100, 10); len(found) != 0 {
			t.Errorf("a run past the end was lit at %+v", found)
		}
	})

	t.Run("a reading that is gone", func(t *testing.T) {
		u, index, _, _, shelved := placing(t, tiny)
		holds(t, index, shelved, "ocr", "abc123")

		if found := litOn(t, u, documentPath, 0, 10); len(found) != 0 {
			t.Errorf("a reading nothing kept was lit at %+v", found)
		}
	})
}

func TestAFileRewrittenSinceItWasReadIsLitFromItself(t *testing.T) {
	// A reading is of the bytes that were there when it was made, and its
	// coordinates describe those. Put against a file rewritten since, they fall
	// where those words no longer are, and nothing on the page says so.
	ctx := t.Context()
	u, index, store, book, shelved := placing(t, outline)
	holds(t, index, shelved, "ocr", "abc123")

	// A reading whose words sit at the top of the first page.
	if err := store.Write(ctx, text.Boxes("ocr", "abc123"), highlight.Pack([]highlight.Box{
		testsupport.Box(0, 0, 400, highlight.Rect{MinX: 0.1, MinY: 0.1, MaxX: 0.9, MaxY: 0.2}),
	})); err != nil {
		t.Fatal(err)
	}
	start, length := run(t, book, "Afterword")

	lit := litOn(t, u, documentPath, start, length)
	if len(lit) == 0 {
		t.Fatal("the reading lit nothing, so rewriting the file proves nothing")
	}

	// The same document, written again.
	shelved.hold(documentPath, domain.KindBook, printedAs(outline), 2)

	after := litOn(t, u, documentPath, start, length)
	if len(after) == 1 && len(after[0].Rects) == 1 && after[0].Rects[0].MinY == 0.1 {
		t.Error("the reading of other bytes was lit on the file that is there now")
	}
}

// A run of a proofread reading is lit where its words now stand. A correction
// that changes a line's length moves everything after it, and the coordinates
// answer about the text the chunks are places in.
func TestAProofreadReadingIsLitWhereItsWordsNowStand(t *testing.T) {
	u, index, store, _, shelved := placing(t, tiny)
	holds(t, index, shelved, "ocr", "abc123")

	written := []highlight.Box{
		testsupport.Box(4, 0, 5, highlight.Rect{MinX: 0.1, MinY: 0.2, MaxX: 0.2, MaxY: 0.23}),
		testsupport.Box(4, 6, 4, highlight.Rect{MinX: 0.21, MinY: 0.2, MaxX: 0.3, MaxY: 0.23}),
		testsupport.Box(5, 11, 5, highlight.Rect{MinX: 0.1, MinY: 0.5, MaxX: 0.2, MaxY: 0.53}),
	}
	if err := store.Write(t.Context(), text.Boxes("ocr", "abc123"), highlight.Pack(written)); err != nil {
		t.Fatal(err)
	}
	// The first line is put right and grows by three bytes, so the third line
	// begins at 14 and runs to 19.
	put := []correction.Line{{Number: 0, Text: "eighteen"}}
	if err := store.Write(t.Context(), text.Corrections("ocr", "abc123"), correction.Pack(put)); err != nil {
		t.Fatal(err)
	}

	found := litOn(t, u, documentPath, 16, 3)
	if len(found) != 1 || found[0].Index != 5 || len(found[0].Rects) != 1 {
		t.Fatalf("the run was lit at %+v", found)
	}
}
