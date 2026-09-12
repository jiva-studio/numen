package epub_test

import (
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/jiva-studio/numen/modules/libs/core/internal/epub"
)

// A window puts this markup on a page unescaped, so a book that writes markup
// of its own — in its text, in an attribute value, in what a picture is
// described as — writes text and nothing else.
func TestABooksOwnMarkupIsText(t *testing.T) {
	book := read(t, spined(t, oneDocumentOpf, map[string]string{
		"OEBPS/one.xhtml": `<html><body>
			<p>Alpha &lt;/p&gt;&lt;script&gt;alert(1)&lt;/script&gt; omega</p>
			<p title='he said "yes" &gt; no'>Beta</p>
			<p><img src="plate.png" alt='&lt;/img&gt;&lt;script&gt;alert(2)&lt;/script&gt;'/></p>
		</body></html>`,
		"OEBPS/plate.png": "\x89PNG\r\n\x1a\n",
	}))
	drawn, err := book.Markup("OEBPS/one.xhtml")
	if err != nil {
		t.Fatalf("markup: %v", err)
	}
	page := drawn.HTML()

	for _, written := range []string{"<script", "alert(1)</", "alert(2)</"} {
		if strings.Contains(page, written) {
			t.Errorf("the markup carries %q:\n%s", written, page)
		}
	}
	root := parsed(t, page)
	if found := elements(root, "script"); len(found) > 0 {
		t.Errorf("a script element was written:\n%s", page)
	}
	// What the book wrote is the book's words, and a reader sees them as words.
	if said := text(root); !strings.Contains(said, "</p><script>alert(1)</script>") {
		t.Errorf("the book's own markup is not among its words: %q", said)
	}
	for _, one := range []struct {
		name, attribute, want string
	}{
		{"p", "title", `he said "yes" > no`},
		{"img", "alt", "</img><script>alert(2)</script>"},
	} {
		got := ""
		for _, found := range elements(root, one.name) {
			if said := valued(found, one.attribute); said != "" {
				got = said
			}
		}
		if got != one.want {
			t.Errorf("%s carries %s=%q, want %q", one.name, one.attribute, got, one.want)
		}
	}
}

// The markup a window is handed says every offset it needs, so the window reads
// them off it and counts none itself.
func TestEveryRunOfTextSaysWhereItStands(t *testing.T) {
	book := read(t, tinyBook(t, map[string]string{secondDoc: "variant/drawn.xhtml"}))
	drawn, err := book.Markup(secondDoc)
	if err != nil {
		t.Fatalf("markup: %v", err)
	}

	runs := offsets(t, parsed(t, drawn.HTML()))
	if len(runs) == 0 {
		t.Fatal("the markup says no offsets")
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

// A picture the book carries is drawn from the entry of the archive it names,
// and an entry the archive does not hold is not there.
func TestAnEntryOfTheArchive(t *testing.T) {
	const plate = "\x89PNG\r\n\x1a\nand the rest of it"
	book := read(t, spined(t, oneDocumentOpf, map[string]string{
		"OEBPS/one.xhtml":   `<html><body><p><img src="plate.png" alt="A plate"/></p></body></html>`,
		"OEBPS/plate.png":   plate,
		"OEBPS/notes.xhtml": `<html><body><p>Beside the spine</p></body></html>`,
	}))

	raw, err := book.Entry("OEBPS/plate.png")
	if err != nil {
		t.Fatalf("the picture the book draws: %v", err)
	}
	if string(raw) != plate {
		t.Errorf("the entry is %q", raw)
	}
	// Every entry is answered for, whatever the spine names: what a picture is
	// filed under is the archive's business.
	if _, err := book.Entry("OEBPS/notes.xhtml"); err != nil {
		t.Errorf("an entry beside the spine: %v", err)
	}
	for _, name := range []string{"", "OEBPS/gone.png", "../outside.png"} {
		if _, err := book.Entry(name); err == nil {
			t.Errorf("%q was answered with bytes", name)
		}
	}
}

// A run of text as the markup carries it.
type run struct {
	at   int
	said string
}

// offsets are the runs of text the markup says, in the order the markup writes
// them.
func offsets(t *testing.T, root *html.Node) []run {
	t.Helper()
	var out []run
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			// A book writes spans of its own, and the run is the one that says
			// where it stands.
			if said := valued(c, epub.OffsetAttribute); said != "" {
				at, err := strconv.Atoi(said)
				if err != nil {
					t.Fatalf("a run says it stands at %q", said)
				}
				out = append(out, run{at: at, said: text(c)})
				continue
			}
			walk(c)
		}
	}
	walk(root)
	return out
}

func parsed(t *testing.T, page string) *html.Node {
	t.Helper()
	root, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatalf("the markup does not parse: %v", err)
	}
	return root
}

// elements are the elements of a name a parsed page holds, in reading order.
func elements(n *html.Node, name string) []*html.Node {
	var out []*html.Node
	if n.Type == html.ElementNode && n.Data == name {
		out = append(out, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = append(out, elements(c, name)...)
	}
	return out
}

// text is what a parsed page says.
func text(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var out strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out.WriteString(text(c))
	}
	return out.String()
}

func valued(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}
