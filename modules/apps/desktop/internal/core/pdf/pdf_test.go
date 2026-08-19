package pdf_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/pdf"
)

// The fixtures are five documents, and each is one thing a reader has to get
// right.
//
// Four of them are written out by hand, uncompressed and without a cross
// reference table, which is what a great many files in the world look like
// after a tool has half-written them: the library rebuilds what it needs, and a
// person can read the fixture. `outline.pdf` is the exception. An outline's
// destinations resolve only in a file whose cross reference table is real, so
// that one is produced rather than typed, and it is still plain text.
const (
	tiny    = "tiny.pdf"    // two pages of text, naming nothing
	outline = "outline.pdf" // four pages, three of them named, one name inside another
	labels  = "labels.pdf"  // front matter numbered apart from the body
	scan    = "scan.pdf"    // two pages that carry no text at all
	prose   = "prose.txt"   // not a document
)

func read(t *testing.T, name string) *pdf.Book {
	t.Helper()
	book, err := pdf.Read(fixture(t, name))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return book
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestPagesAreTheReadingOrder(t *testing.T) {
	book := read(t, tiny)

	want := "Alpha beta gamma\nDelta epsilon zeta\n"
	if book.Text != want {
		t.Errorf("text is %q, want %q", book.Text, want)
	}
	if len(book.Pages) != 2 {
		t.Fatalf("%d pages, want 2", len(book.Pages))
	}
	if book.Pages[0].Offset != 0 {
		t.Errorf("the first page begins at %d, want 0", book.Pages[0].Offset)
	}
	if at := book.Pages[1].Offset; !strings.HasPrefix(book.Text[at:], "Delta") {
		t.Errorf("the second page begins at %d, which is %q", at, book.Text[at:])
	}
}

func TestEveryPageIsNamed(t *testing.T) {
	// A document that numbers its front matter apart from its body says so, and
	// what it says is what a person holding the printed book would say.
	book := read(t, labels)

	want := []string{"i", "ii", "1"}
	if len(book.Pages) != len(want) {
		t.Fatalf("%d pages, want %d", len(book.Pages), len(want))
	}
	for i, label := range want {
		if book.Pages[i].Label != label {
			t.Errorf("page %d is called %q, want %q", i, book.Pages[i].Label, label)
		}
	}

	// A document naming none of its pages still names all of them: the number
	// of the page is what it is called.
	plain := read(t, tiny)
	for i, page := range plain.Pages {
		if want := []string{"1", "2"}[i]; page.Label != want {
			t.Errorf("page %d is called %q, want %q", i, page.Label, want)
		}
	}
}

func TestStructureComesFromTheOutlineOrFromNothing(t *testing.T) {
	tests := []struct {
		name   string
		file   string
		want   pdf.Structure
		places []string
	}{
		{
			name:   "an outline names the parts",
			file:   outline,
			want:   pdf.FromOutline,
			places: []string{"The First Part", "A Closer Reading", "Afterword"},
		},
		{
			name: "a document with no outline names nothing",
			file: tiny,
			want: pdf.FromNothing,
		},
		{
			name: "a scan names nothing either",
			file: scan,
			want: pdf.FromNothing,
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			book := read(t, c.file)
			if book.Structure != c.want {
				t.Errorf("structure is %q, want %q", book.Structure, c.want)
			}
			var named []string
			for _, p := range book.Places {
				named = append(named, p.Title)
			}
			if len(named) != len(c.places) {
				t.Fatalf("places are %q, want %q", named, c.places)
			}
			for i := range named {
				if named[i] != c.places[i] {
					t.Errorf("place %d is %q, want %q", i, named[i], c.places[i])
				}
			}
		})
	}
}

func TestAPlaceCarriesTheDepthTheOutlineGaveIt(t *testing.T) {
	book := read(t, outline)

	want := []int{1, 2, 1}
	for i, level := range want {
		if book.Places[i].Level != level {
			t.Errorf("%q is at level %d, want %d", book.Places[i].Title, book.Places[i].Level, level)
		}
	}
}

func TestAPlaceBeginsWhereItsPageBegins(t *testing.T) {
	book := read(t, outline)

	for _, place := range book.Places {
		at := -1
		for _, page := range book.Pages {
			if page.Offset == place.Offset {
				at = page.Offset
			}
		}
		if at < 0 {
			t.Errorf("%q begins at %d, which is not where any page begins", place.Title, place.Offset)
		}
	}
}

func TestAScanIsReadAndSaysNothing(t *testing.T) {
	// A document with no text layer is not a failure. It is read, its pages are
	// counted, and what it says is nothing — which is what makes it a scan and
	// what a reader has to be able to tell.
	book := read(t, scan)

	if book.Text != "" {
		t.Errorf("text is %q, want it empty", book.Text)
	}
	if len(book.Pages) != 2 {
		t.Errorf("%d pages, want 2", len(book.Pages))
	}
}

func TestLocate(t *testing.T) {
	book := read(t, outline)

	// Keyed by a word in the text, so that the case says where it is asking
	// about rather than what the offset happens to be.
	at := func(word string) int {
		t.Helper()
		i := strings.Index(book.Text, word)
		if i < 0 {
			t.Fatalf("the fixture does not say %q", word)
		}
		return i
	}

	tests := []struct {
		name  string
		at    int
		place string
		page  string
	}{
		{name: "before every name", at: at("Front"), place: "", page: "1"},
		{name: "at a name", at: at("Opening"), place: "The First Part", page: "2"},
		{name: "inside a nested name", at: at("closer"), place: "A Closer Reading", page: "3"},
		{name: "after the last name", at: at("Afterword"), place: "Afterword", page: "4"},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			where := book.Locate(c.at)
			if where.Place != c.place {
				t.Errorf("place is %q, want %q", where.Place, c.place)
			}
			if where.Page != c.page {
				t.Errorf("page is %q, want %q", where.Page, c.page)
			}
		})
	}
}

func TestTheSameBytesGiveTheSameText(t *testing.T) {
	// A chunk keeps an offset and not the text, so a second reading of the same
	// file has to agree with the first.
	raw := fixture(t, outline)

	first, err := pdf.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := pdf.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	if first.Text != second.Text {
		t.Error("the same file read twice gives two texts")
	}
	if len(first.Places) != len(second.Places) {
		t.Fatalf("the same file names %d places and then %d", len(first.Places), len(second.Places))
	}
	for i := range first.Places {
		if first.Places[i] != second.Places[i] {
			t.Errorf("place %d is %+v and then %+v", i, first.Places[i], second.Places[i])
		}
	}
}

func TestWhatIsNotADocument(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		want error
	}{
		{name: "prose", raw: fixture(t, prose), want: pdf.ErrNotPDF},
		{name: "nothing at all", raw: nil, want: pdf.ErrNotPDF},
		{name: "a header and no more", raw: []byte("%PDF-1.4\n"), want: pdf.ErrNotPDF},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			book, err := pdf.Read(c.raw)
			if !errors.Is(err, c.want) {
				t.Fatalf("error is %v, want %v", err, c.want)
			}
			if book != nil {
				t.Error("a file that is not a document was read into one")
			}
		})
	}
}

func TestReadingIsSafeFromSeveralGoroutines(t *testing.T) {
	// A vault is walked by one goroutine today and by more as soon as anything
	// asks it to be, and the library holds a document per worker.
	raw := fixture(t, tiny)

	done := make(chan string, 8)
	for i := 0; i < cap(done); i++ {
		go func() {
			book, err := pdf.Read(raw)
			if err != nil {
				done <- "error: " + err.Error()
				return
			}
			done <- book.Text
		}()
	}
	want := read(t, tiny).Text
	for i := 0; i < cap(done); i++ {
		if got := <-done; got != want {
			t.Errorf("one reading gave %q, want %q", got, want)
		}
	}
}
