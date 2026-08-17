package epub_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/epub"
)

// The fixture is a book with everything wrong with it that a real book has: a
// document with three opening body tags, a manifest naming a file the archive
// does not hold, a spine naming an item nothing declares, and a navigation
// control file whose one entry is the cover.
const (
	coverDoc  = "OEBPS/cover.xhtml"
	firstDoc  = "OEBPS/first.xhtml"
	secondDoc = "OEBPS/second.xhtml"
)

func TestTheSpineIsTheReadingOrder(t *testing.T) {
	raw := tinyBook(t, nil)
	book := read(t, raw)

	want := []string{coverDoc, firstDoc, secondDoc}
	var got []string
	for _, doc := range book.Documents {
		got = append(got, doc.Path)
	}
	if !slices.Equal(got, want) {
		t.Errorf("documents = %v, want %v", got, want)
	}

	// The archive is written in another order on purpose: where a file sits in
	// the zip says nothing about where it is read.
	if archived := archiveOrder(t, raw); slices.Equal(intersect(archived, want), want) {
		t.Fatalf("the fixture archives the documents in reading order (%v), so this test cannot fail", archived)
	}

	for _, word := range []string{"Cover", "Alpha", "Delta"} {
		if !strings.Contains(book.Text, word) {
			t.Fatalf("%q is missing from the text", word)
		}
	}
	if !(strings.Index(book.Text, "Cover") < strings.Index(book.Text, "Alpha") &&
		strings.Index(book.Text, "Alpha") < strings.Index(book.Text, "Delta")) {
		t.Errorf("the text is not in spine order:\n%s", book.Text)
	}

	for i, doc := range book.Documents {
		if doc.Offset+doc.Length > len(book.Text) {
			t.Errorf("document %d runs past the text", i)
		}
		if i > 0 && book.Documents[i-1].Offset >= doc.Offset {
			t.Errorf("document %d does not begin after the one before it", i)
		}
	}
}

func TestMalformedMarkupIsRead(t *testing.T) {
	book := read(t, tinyBook(t, nil))

	// One paragraph per body tag, and the last one closes nothing.
	for _, word := range []string{"Alpha", "Beta", "Gamma"} {
		if !strings.Contains(book.Text, word) {
			t.Errorf("%q is missing: the second and third body tags were not read", word)
		}
	}
}

func TestWhatIsNotTextOfTheBook(t *testing.T) {
	book := read(t, tinyBook(t, nil))

	if strings.Contains(book.Text, "forbidden") {
		t.Errorf("a script or a stylesheet reached the text:\n%s", book.Text)
	}
	if strings.Contains(book.Text, "Second") {
		t.Errorf("the title element of a document reached the text:\n%s", book.Text)
	}
}

func TestADocumentTheArchiveDoesNotHoldIsSkipped(t *testing.T) {
	book := read(t, tinyBook(t, nil))

	for _, doc := range book.Documents {
		if strings.Contains(doc.Path, "gone") {
			t.Errorf("a document the archive does not hold was kept: %s", doc.Path)
		}
	}
	if len(book.Documents) != 3 {
		t.Errorf("documents = %d, want the three the archive holds", len(book.Documents))
	}
}

func TestTheTitleDoesNotDependOnThePrefix(t *testing.T) {
	// The fixture writes its title in the Dublin Core namespace under a prefix
	// of its own invention.
	if got := read(t, tinyBook(t, nil)).Title; got != "A Tiny Book" {
		t.Errorf("title = %q", got)
	}
}

func TestStructureFallsThroughTheTiers(t *testing.T) {
	plain := map[string]string{
		firstDoc:  "variant/plain.xhtml",
		secondDoc: "variant/plain.xhtml",
	}
	chapters := map[string]string{"OEBPS/toc.ncx": "variant/chapters.ncx"}
	navigation := map[string]string{
		"OEBPS/content.opf": "variant/nav.opf",
		"OEBPS/nav.xhtml":   "variant/nav.xhtml",
	}

	for _, c := range []struct {
		name    string
		replace map[string]string
		want    epub.Structure
		places  []string
	}{{
		name:    "a control file that names the parts answers",
		replace: chapters,
		want:    epub.FromNavigation,
		places:  []string{"Of the Beginning", "Alpha", "Of the Middle"},
	}, {
		name:    "an EPUB 3 navigation document answers when the control file names only the cover",
		replace: navigation,
		want:    epub.FromNavigation,
		places:  []string{"Of the Beginning", "Of the Middle"},
	}, {
		name:    "headings answer when no navigation document does",
		replace: nil,
		want:    epub.FromHeadings,
		places:  []string{"Of the Beginning", "Of the Middle"},
	}, {
		name:    "a book that names nothing is a book",
		replace: plain,
		want:    epub.FromNothing,
		places:  nil,
	}} {
		t.Run(c.name, func(t *testing.T) {
			book := read(t, tinyBook(t, c.replace))

			if book.Structure != c.want {
				t.Errorf("structure = %q, want %q (places: %v)", book.Structure, c.want, titles(book))
			}
			if got := titles(book); !slices.Equal(got, c.places) {
				t.Errorf("places = %v, want %v", got, c.places)
			}
			for _, place := range book.Places {
				if !strings.HasPrefix(book.Text[place.Offset:], place.Title) {
					t.Errorf("place %q does not begin at its offset %d: %q",
						place.Title, place.Offset, excerpt(book.Text, place.Offset))
				}
			}
		})
	}
}

func TestHeadingsAreReadAtAnyLevel(t *testing.T) {
	book := read(t, tinyBook(t, nil))

	// The fixture heads its sections with h4. The level a book chose is kept and
	// says nothing about whether the heading counts.
	for _, place := range book.Places {
		if place.Level != 4 {
			t.Errorf("place %q is at level %d, want the level the markup wrote", place.Title, place.Level)
		}
	}
}

func TestPrintedPages(t *testing.T) {
	t.Run("from the markup", func(t *testing.T) {
		book := read(t, tinyBook(t, nil))

		if len(book.Pages) != 1 || book.Pages[0].Label != "42" {
			t.Fatalf("pages = %v, want the one the markup marks", book.Pages)
		}
		at := strings.Index(book.Text, "Epsilon")
		if got := book.Locate(at).Page; got != "42" {
			t.Errorf("page at Epsilon = %q, want 42", got)
		}
		if got := book.Locate(strings.Index(book.Text, "Alpha")).Page; got != "" {
			t.Errorf("page before the first page break = %q, want none", got)
		}
	})

	t.Run("from the page list", func(t *testing.T) {
		book := read(t, tinyBook(t, map[string]string{"OEBPS/toc.ncx": "variant/chapters.ncx"}))

		// The page list names the page by its value, not by the decoration
		// around it.
		if len(book.Pages) != 1 || book.Pages[0].Label != "7" {
			t.Fatalf("pages = %v, want the one the page list names", book.Pages)
		}
	})

	t.Run("a book with none", func(t *testing.T) {
		book := read(t, tinyBook(t, map[string]string{secondDoc: "variant/plain.xhtml"}))

		if len(book.Pages) != 0 {
			t.Errorf("pages = %v, want none", book.Pages)
		}
	})
}

func TestLocate(t *testing.T) {
	book := read(t, tinyBook(t, nil))
	at := func(word string) int {
		t.Helper()
		i := strings.Index(book.Text, word)
		if i < 0 {
			t.Fatalf("%q is not in the text", word)
		}
		return i
	}

	for _, c := range []struct {
		name     string
		offset   int
		document string
		place    string
		page     string
	}{
		{name: "the first byte of the book", offset: 0, document: coverDoc},
		{name: "before the first heading", offset: at("Cover"), document: coverDoc},
		{name: "inside the first section", offset: at("Gamma"), document: firstDoc, place: "Of the Beginning"},
		{name: "inside the second section", offset: at("Delta"), document: secondDoc, place: "Of the Middle"},
		{name: "after the page break", offset: at("Epsilon"), document: secondDoc, place: "Of the Middle", page: "42"},
		{name: "the last byte", offset: len(book.Text) - 1, document: secondDoc, place: "Of the Middle", page: "42"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := book.Locate(c.offset)
			if got.Document != c.document {
				t.Errorf("document = %q, want %q", got.Document, c.document)
			}
			if got.Place != c.place {
				t.Errorf("place = %q, want %q", got.Place, c.place)
			}
			if got.Page != c.page {
				t.Errorf("page = %q, want %q", got.Page, c.page)
			}
			if got.Place != "" && got.PlaceOffset > c.offset {
				t.Errorf("place offset %d is after the offset asked about", got.PlaceOffset)
			}
		})
	}
}

func TestTheSameBytesGiveTheSameText(t *testing.T) {
	// A chunk keeps an offset and not the text, so a second reading of the same
	// file has to agree with the first.
	raw := tinyBook(t, nil)
	first, second := read(t, raw), read(t, raw)

	if first.Text != second.Text {
		t.Error("the text differs between two readings")
	}
	if !slices.Equal(first.Documents, second.Documents) {
		t.Errorf("documents differ: %v and %v", first.Documents, second.Documents)
	}
	if !slices.Equal(first.Places, second.Places) {
		t.Errorf("places differ: %v and %v", first.Places, second.Places)
	}
}

func TestWhatIsNotABook(t *testing.T) {
	for _, c := range []struct {
		name string
		raw  []byte
		want error
	}{{
		name: "not an archive",
		raw:  []byte("PK, and then a lie about what follows"),
		want: epub.ErrNotArchive,
	}, {
		name: "no container",
		raw:  tinyBook(t, nil, "META-INF/container.xml"),
		want: epub.ErrNoContainer,
	}, {
		name: "the package document is not in the archive",
		raw:  tinyBook(t, nil, "OEBPS/content.opf"),
		want: epub.ErrNoPackage,
	}, {
		name: "the package document does not parse",
		raw:  tinyBook(t, map[string]string{"OEBPS/content.opf": "variant/garbage.opf"}),
		want: epub.ErrNoPackage,
	}} {
		t.Run(c.name, func(t *testing.T) {
			book, err := epub.Read(c.raw)
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if book != nil {
				t.Errorf("a book was returned with an error")
			}
		})
	}
}

func read(t *testing.T, raw []byte) *epub.Book {
	t.Helper()
	book, err := epub.Read(raw)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return book
}

func titles(book *epub.Book) []string {
	var out []string
	for _, place := range book.Places {
		out = append(out, place.Title)
	}
	return out
}

func excerpt(text string, at int) string {
	end := min(at+40, len(text))
	return text[at:end]
}

// tinyBook zips the committed fixture into an EPUB.
//
// replace maps a path in the archive to the fixture file that fills it, adding
// the path when the fixture has no such file; omit drops paths.
func tinyBook(t *testing.T, replace map[string]string, omit ...string) []byte {
	t.Helper()
	root := filepath.Join("testdata", "tiny")

	parts := map[string][]byte{}
	err := filepath.WalkDir(root, func(at string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		raw, err := os.ReadFile(at)
		if err != nil {
			return err
		}
		name, err := filepath.Rel(root, at)
		if err != nil {
			return err
		}
		parts[filepath.ToSlash(name)] = raw
		return nil
	})
	if err != nil {
		t.Fatalf("read the fixture: %v", err)
	}
	for at, from := range replace {
		raw, err := os.ReadFile(filepath.Join("testdata", filepath.FromSlash(from)))
		if err != nil {
			t.Fatalf("read the fixture: %v", err)
		}
		parts[at] = raw
	}
	for _, at := range omit {
		delete(parts, at)
	}

	// The mimetype comes first because the specification asks for it. The rest
	// are written in reverse, so that the order of the archive is not the order
	// of the book.
	names := slices.Sorted(maps.Keys(parts))
	slices.Reverse(names)
	if i := slices.Index(names, "mimetype"); i >= 0 {
		names = append([]string{"mimetype"}, slices.Delete(names, i, i+1)...)
	}

	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	for _, name := range names {
		method := zip.Deflate
		if name == "mimetype" {
			method = zip.Store
		}
		entry, err := archive.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		if err != nil {
			t.Fatalf("write the archive: %v", err)
		}
		if _, err := entry.Write(parts[name]); err != nil {
			t.Fatalf("write the archive: %v", err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("write the archive: %v", err)
	}
	return out.Bytes()
}

// archiveOrder is the order the entries were written in.
func archiveOrder(t *testing.T, raw []byte) []string {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("open the archive: %v", err)
	}
	var names []string
	for _, f := range archive.File {
		names = append(names, f.Name)
	}
	return names
}

// intersect keeps the members of names that are wanted, in the order of names.
func intersect(names, wanted []string) []string {
	var out []string
	for _, name := range names {
		if slices.Contains(wanted, name) {
			out = append(out, name)
		}
	}
	return out
}
