package epub_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/epub"
)

// The corpus is forty EPUB files of the Mahābhārata, in four languages and from
// a dozen converters. It is not in the repository: it is read where it is, and
// the test stands aside when it is not there.
const corpusEnv = "NUMEN_EPUB_CORPUS"

const relativeCorpus = "../../../../../../resources/mahabharata"

// What the corpus holds, measured. These numbers are the test: a change that
// moves a book from one tier to another has changed what a reader is shown, and
// has to be looked at.
//
// Half the corpus names nothing: sixteen of those twenty books carry no heading
// at all, and four carry one heading, which names the book.
var wantStructure = map[epub.Structure]int{
	epub.FromNavigation: 16,
	epub.FromHeadings:   4,
	epub.FromNothing:    20,
}

const (
	wantBooks = 40
	// The corpus yields 58.7 MB of text. The bounds are wide because what is
	// checked here is that every book was read, not how spaces were counted.
	leastText   = 50_000_000
	mostText    = 70_000_000
	leastInBook = 1_000
)

func TestTheCorpusIsRead(t *testing.T) {
	books := corpus(t)
	if len(books) != wantBooks {
		t.Errorf("the corpus holds %d books, want %d", len(books), wantBooks)
	}

	structure := map[epub.Structure]int{}
	total := 0
	for _, at := range books {
		t.Run(filepath.Base(at), func(t *testing.T) {
			raw, err := os.ReadFile(at)
			if err != nil {
				t.Fatalf("read the file: %v", err)
			}
			// A book that is merely poor is still read. Every one of these is
			// poor in some way, and none of them may fail.
			book, err := epub.Read(raw)
			if err != nil {
				t.Fatalf("read the book: %v", err)
			}
			structure[book.Tier]++
			total += len(book.Text)

			checkBook(t, book)
		})
	}

	t.Logf("structure: %d navigation, %d headings, %d none; %d bytes of text",
		structure[epub.FromNavigation], structure[epub.FromHeadings],
		structure[epub.FromNothing], total)

	for _, tier := range []epub.Structure{epub.FromNavigation, epub.FromHeadings, epub.FromNothing} {
		if structure[tier] != wantStructure[tier] {
			t.Errorf("%s answered for %d books, want %d", tier, structure[tier], wantStructure[tier])
		}
	}
	if total < leastText || total > mostText {
		t.Errorf("the corpus yields %d bytes of text, want between %d and %d", total, leastText, mostText)
	}
}

// checkBook holds every book to what a caller is promised: the text is the
// documents in order, and every offset lands where it says.
func checkBook(t *testing.T, book *epub.Book) {
	t.Helper()

	if len(book.Text) < leastInBook {
		t.Errorf("text = %d bytes, want a book's worth", len(book.Text))
	}
	if len(book.Documents) == 0 {
		t.Fatal("no spine document was read")
	}

	for i, doc := range book.Documents {
		if i > 0 && book.Documents[i-1].Offset > doc.Offset {
			t.Fatalf("document %d (%s) begins before the one before it", i, doc.Path)
		}
		if doc.Offset+doc.Length > len(book.Text) {
			t.Fatalf("document %d (%s) runs past the text", i, doc.Path)
		}
		// A spine document may carry no text at all: a title page is often one
		// image. It holds no offset, so nothing is located in it.
		if doc.Length == 0 {
			continue
		}
		if got := book.Locate(doc.Offset).Document; got != doc.Path {
			t.Fatalf("the first byte of %s is located in %s", doc.Path, got)
		}
	}

	for i, part := range book.Parts {
		if part.Offset < 0 || part.Offset > len(book.Text) {
			t.Fatalf("part %q is at %d, outside the text", part.Title, part.Offset)
		}
		if i > 0 && book.Parts[i-1].Offset > part.Offset {
			t.Fatalf("part %q is out of order", part.Title)
		}
		if book.Tier == epub.FromHeadings && !strings.HasPrefix(book.Text[part.Offset:], part.Title) {
			t.Fatalf("heading %q does not begin at its offset %d", part.Title, part.Offset)
		}
		at := book.Locate(part.Offset)
		if at.PartOffset != part.Offset {
			t.Fatalf("part %q at %d is located at %d", part.Title, part.Offset, at.PartOffset)
		}
	}

	for i, page := range book.Pages {
		if page.Offset < 0 || page.Offset > len(book.Text) {
			t.Fatalf("page %q is at %d, outside the text", page.Label, page.Offset)
		}
		if i > 0 && book.Pages[i-1].Offset > page.Offset {
			t.Fatalf("page %q is out of order", page.Label)
		}
	}

	if book.Tier == epub.FromNothing && len(book.Parts) != 0 {
		t.Errorf("a book that names nothing came back with %d parts", len(book.Parts))
	}

	checkPages(t, book)
	checkMarkup(t, book)
}

// checkPages holds every page of a book to beginning at a letter and on the page
// it says. Half this corpus is written in Devanagari or Cyrillic, where a letter
// is two or three bytes, so a page counted in bytes alone opens in the middle of
// one.
func checkPages(t *testing.T, book *epub.Book) {
	t.Helper()

	for page := 1; page <= book.PageCount(); page++ {
		at := book.PageStart(page)
		if r, _ := utf8.DecodeRuneInString(book.Text[at:]); r == utf8.RuneError {
			t.Fatalf("page %d of %d begins at %d, in the middle of a letter",
				page, book.PageCount(), at)
		}
		if got := book.PageOf(at); got != page {
			t.Fatalf("page %d begins at %d, which is on page %d", page, at, got)
		}
	}
}

// checkMarkup holds every document to the one property a reader is built on: the
// words of the markup are the words of the text, at the offsets the text is
// chunked and searched at. A reader takes an offset off an attribute and never
// counts one out itself, so a document where the two differ by a byte lights the
// wrong words.
func checkMarkup(t *testing.T, book *epub.Book) {
	t.Helper()

	for _, doc := range book.Documents {
		drawn, err := book.Markup(doc.Path)
		if err != nil {
			t.Fatalf("markup of %s: %v", doc.Path, err)
		}
		if drawn.Offset != doc.Offset || drawn.Length != doc.Length {
			t.Fatalf("markup of %s runs from %d for %d, and the document from %d for %d",
				doc.Path, drawn.Offset, drawn.Length, doc.Offset, doc.Length)
		}
		want := book.Text[doc.Offset : doc.Offset+doc.Length]
		if got := said(drawn.Nodes); got != want {
			t.Fatalf("the markup of %s says %d bytes and its text is %d:\n%q\n%q",
				doc.Path, len(got), len(want), excerpt(got, 0), excerpt(want, 0))
		}
		checkSpans(t, book.Text, doc, drawn.Nodes, doc.Offset+doc.Length)
		checkHTML(t, book, doc, drawn)
	}
}

// checkSpans holds every node to the run of text it stands over. A node reaches
// to where the next one begins, and what it and everything under it says is that
// run of the book's text, byte for byte: a word that slipped out of the element
// it was written in is a word a reader draws in the wrong place.
func checkSpans(t *testing.T, text string, doc epub.Document, nodes []epub.Node, limit int) {
	t.Helper()
	for i, node := range nodes {
		to := limit
		if i+1 < len(nodes) {
			to = nodes[i+1].Offset
		}
		if node.Offset < doc.Offset || node.Offset > to || to > len(text) {
			t.Fatalf("a node of %s runs from %d to %d, outside the document at %d for %d",
				doc.Path, node.Offset, to, doc.Offset, doc.Length)
		}
		if got := said([]epub.Node{node}); got != text[node.Offset:to] {
			t.Fatalf("a %q of %s says %q and stands over %q",
				node.Name, doc.Path, got, text[node.Offset:to])
		}
		checkSpans(t, text, doc, node.Children, to)
	}
}

// said is the text a document's markup carries, in reading order.
func said(nodes []epub.Node) string {
	var out strings.Builder
	var walk func([]epub.Node)
	walk = func(nodes []epub.Node) {
		for _, node := range nodes {
			out.WriteString(node.Text)
			walk(node.Children)
		}
	}
	walk(nodes)
	return out.String()
}

// corpus is where the books are, or a reason to stand aside.
func corpus(t *testing.T) []string {
	t.Helper()
	root := os.Getenv(corpusEnv)
	if root == "" {
		root = relativeCorpus
	}
	if _, err := os.Stat(root); err != nil {
		t.Skipf("the corpus is not here: %v (set %s to say where it is)", err, corpusEnv)
	}

	books, err := booksUnder(root)
	if err != nil {
		t.Fatalf("walk the corpus: %v", err)
	}
	if len(books) == 0 {
		t.Skipf("the corpus at %s holds no books", root)
	}
	return books
}

// booksUnder is every EPUB file under a folder.
func booksUnder(root string) ([]string, error) {
	var books []string
	err := filepath.WalkDir(root, func(at string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(at), ".epub") {
			books = append(books, at)
		}
		return nil
	})
	return books, err
}
