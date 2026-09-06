package epub_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/jiva-studio/numen/modules/libs/core/epub"
)

// update writes the spine document down again, and compares against none of it.
var update = flag.Bool("update", false, "write the spine document of the corpus again")

// spineCorpus is one spine document as this package emits it, which the window
// that draws it is tested against. It stands with the corpora the two sides
// share, because it is the one thing about a book that crosses as neither the
// schema nor bytes: an attribute name, a set of elements, an escaping and a set
// of offsets, agreed on by a producer here and a reader there.
const spineCorpus = "../../protocol/testdata/spine.html"

// middleDoc is the document the corpus is taken from. A document stands before
// it in the spine, so its offsets begin where nothing else does.
const middleDoc = "OEBPS/middle.xhtml"

// A change meaning to move the markup moves this file with it, under -update,
// and the diff is what it moved.
func TestASpineDocumentCrossesAsTheCorpusSaysItDoes(t *testing.T) {
	got := seamMarkup(t).HTML() + "\n"
	if *update {
		if err := os.WriteFile(spineCorpus, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(spineCorpus)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("the document crosses as\n%s\nand the corpus says\n%s", got, want)
	}
}

// Every offset the corpus carries is where that run stands in the book's text,
// so the window reading them off it is reading the book's own numbers.
func TestTheCorpusCarriesTheBooksOwnOffsets(t *testing.T) {
	book := read(t, seamBook(t))
	drawn := seamMarkup(t)

	runs := offsets(t, parsed(t, drawn.HTML()))
	if len(runs) == 0 {
		t.Fatal("the corpus says no offsets")
	}
	if drawn.Offset == 0 {
		t.Error("the corpus is taken from the first document, and its offsets say nothing")
	}
	var out strings.Builder
	for _, run := range runs {
		if !strings.HasPrefix(book.Text[run.at:], run.said) {
			t.Errorf("a run says it begins at %d, where the text is %q",
				run.at, excerpt(book.Text, run.at))
		}
		out.WriteString(run.said)
	}
	if want := book.Text[drawn.Offset : drawn.Offset+drawn.Length]; out.String() != want {
		t.Errorf("the runs say\n%q\nand the document is\n%q", out.String(), want)
	}
}

// A picture is drawn from an entry of the archive, so the address the markup
// names it by is the name that entry is answered under.
func TestAPictureIsNamedByTheEntryItIsDrawnFrom(t *testing.T) {
	book := read(t, seamBook(t))
	drawn := seamMarkup(t)

	named := pictures(parsed(t, drawn.HTML()))
	if len(named) == 0 {
		t.Fatal("the document draws no picture")
	}
	for _, name := range named {
		if _, err := book.Entry(name); err != nil {
			t.Errorf("the markup draws %q, and the archive answers %v", name, err)
		}
	}
}

// A link inside the book names a document of the book, or a place in the
// document being read. Either is a move within the book and never an address
// the window is sent to.
func TestALinkInsideTheBookNamesADocumentOrAPlaceInThisOne(t *testing.T) {
	book := read(t, seamBook(t))
	held := map[string]bool{}
	for _, doc := range book.Documents {
		held[doc.Path] = true
	}

	var within, down, away int
	for _, found := range elements(parsed(t, seamMarkup(t).HTML()), "a") {
		href := valued(found, "href")
		target, fragment, _ := strings.Cut(href, "#")
		switch {
		case strings.Contains(target, ":"):
			away++
		case target != "":
			if !held[target] {
				t.Errorf("a link names %q, and the book is not read in it", target)
			}
			within++
		case fragment != "":
			down++
		default:
			t.Errorf("a link carries %q, which leads nowhere", href)
		}
	}
	if within == 0 || down == 0 || away == 0 {
		t.Errorf("the corpus carries %d links to another document, %d to a place in this one and %d out of the book",
			within, down, away)
	}
}

// seamMarkup is the spine document the corpus is taken from.
func seamMarkup(t *testing.T) *epub.Markup {
	t.Helper()
	drawn, err := read(t, seamBook(t)).Markup(middleDoc)
	if err != nil {
		t.Fatalf("markup: %v", err)
	}
	return drawn
}

// seamBook zips the committed fixture the corpus is drawn from.
func seamBook(t *testing.T) []byte {
	t.Helper()
	return zipped(t, partsIn(t, filepath.Join("testdata", "seam")))
}

// pictures are the entries of the archive a document draws, and none of the
// pictures written into the markup itself.
func pictures(root *html.Node) []string {
	var out []string
	for _, found := range elements(root, "img") {
		if src := valued(found, "src"); src != "" && !strings.HasPrefix(src, "data:") {
			out = append(out, src)
		}
	}
	return out
}
