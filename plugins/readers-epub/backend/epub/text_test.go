package epub_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
)

// The text of a document is read twice, once as the book's text and once as the
// markup a window draws, and both walks say the same words.
func TestMarkupBecomesText(t *testing.T) {
	for _, c := range []struct {
		name, markup, want string
	}{
		{"a space after the first letter of a book stands", `<p>a b</p>`, "a b\n"},
		{"a form feed, a vertical tab and a carriage return are space",
			`<p>a&#12;b&#11;c&#13;d</p>`, "a b c d\n"},
		{"a space outside ASCII is space", "<p>a\u2003b</p>", "a b\n"},
		{"a space outside ASCII at the start of a book is not text", "<p>\u2003a</p>", "a\n"},
		{"a space outside ASCII after a space is not text", "<p>a \u2003b</p>", "a b\n"},
		{"a space outside ASCII at the start of a line is not text", "<p>a</p>\u2003b", "a\nb\n"},
		{"the space before the end of a block is taken back", `<p>Ab </p>`, "Ab\n"},
		{"text standing in no element still ends its line", `Nu`, "Nu\n"},
		{"a block begins a line after the text before it", `<body>Lead<p>Para</p></body>`, "Lead\nPara\n"},
		{"text after a block begins a line", `<body><div>A</div>B</body>`, "A\nB\n"},
		{"a block's end tag that closes nothing ends the line", `<body>A</p>B</body>`, "A\nB\n"},
		{"a line break written as an end tag breaks the line", `<body>A</br>B</body>`, "A\nB\n"},
		{"a line break written as a start tag breaks the line", `<p>A<br/>B</p>`, "A\nB\n"},
		{"text after the body's end tag is in the body", `<html><body>One</body>Two</html>`, "OneTwo\n"},
		{"text after the document's end tag is in the body", `<html><body>One</body></html>Two`, "OneTwo\n"},
		{"an end tag naming no element closes nothing", `<p>v<b>i</bogus>i</b></p>`, "vii\n"},

		{"a pre keeps its spaces", `<pre>a  b</pre>`, "a  b\n"},
		{"the newline right after a pre is not text", "<pre>\na  b</pre>", "a  b\n"},
		{"a carriage return right after a pre is not text", `<pre>&#13;x</pre>`, "x\n"},
		{"a pre holding only a carriage return holds no text", `<p>a</p><pre>&#13;</pre>`, "a\n"},
		{"only the first text of a pre loses its newline", "<pre>\na<!-- c -->\nb</pre>", "a\nb\n"},
		{"a newline after an element inside a pre is text",
			"<p>Lead</p><pre><code>\nx</code></pre>", "Lead\n\nx\n"},
		{"text on either side of a pre is not preformatted",
			`<p>x  y</p><pre>a  b</pre><p>c   d</p>`, "x y\na  b\nc d\n"},
		{"an element closing inside a pre leaves it preformatted",
			`<pre><b>a</b>  b</pre>`, "a  b\n"},
		{"a picture inside a pre leaves it preformatted",
			`<pre>a<img src="p.png"/>  b</pre>`, "a  b\n"},
		{"the newline right after a listing is not text",
			"<body>Lead<listing>\nA</listing></body>", "LeadA\n"},
		{"the newline right after a textarea is not text",
			"<body>Lead<textarea>\nA</textarea></body>", "LeadA\n"},
		{"the newline right after any other element is space",
			"<p>Lead<b>\nx</b></p>", "Lead x\n"},

		{"a style and a title in the body are not text",
			`<body><style>.x{}</style><p>Seen</p><title>Hidden</title><p>Too</p></body>`, "Seen\nToo\n"},
		{"text following a style element in an inline container is preserved",
			`<body><b><style>.x{}</style>Inline</b></body>`, "Inline\n"},
		{"a head left open ends where the body begins",
			`<html><head><meta charset="utf-8"/><body><p>Words</p></body></html>`, "Words\n"},
		{"a head left open ends where a block begins",
			`<html><head><meta charset="utf-8"/><p>Words</p></html>`, "Words\n"},
		{"what a head holds after its title is not text",
			`<html><head><title>T</title><noscript>Hidden</noscript></head><body><p>Seen</p></body></html>`, "Seen\n"},
		{"an unmatched end tag does not close enclosing frames",
			`<div><p>One</p></span><p>Two</p></div>`, "One\nTwo\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			book, markup := readOne(t, c.markup)
			if book.Text != c.want {
				t.Errorf("the text is %q, want %q", book.Text, c.want)
			}
			if got := getNodeText(markup.Nodes); got != c.want {
				t.Errorf("the markup says %q, want %q", got, c.want)
			}
		})
	}
}

// A page of the printed book is marked by an element whose epub:type carries
// the token pagebreak, and is named by its title or by the text it holds.
func TestPagesTheMarkupMarks(t *testing.T) {
	for _, c := range []struct {
		name, markup string
		want         []string
	}{
		{"a page named by the text it holds",
			`<p>Before <span epub:type="pagebreak">xii</span> after</p>`, []string{"xii"}},
		{"a token among others separated by a space",
			`<p><span epub:type="noteref pagebreak" title="2"></span>Two</p>`, []string{"2"}},
		{"a token among others separated by a tab",
			"<p><span epub:type=\"noteref\tpagebreak\" title=\"2\"></span>Two</p>", []string{"2"}},
		{"a token among others separated by a newline",
			"<p><span epub:type=\"noteref\npagebreak\" title=\"2\"></span>Two</p>", []string{"2"}},
		{"a token among others separated by a carriage return",
			`<p><span epub:type="noteref&#13;pagebreak" title="2"></span>Two</p>`, []string{"2"}},
		{"a token after a space",
			`<p><span epub:type=" pagebreak" title="3"></span>Three</p>`, []string{"3"}},
		{"a token after a tab",
			"<p><span epub:type=\"\tpagebreak\" title=\"3\"></span>Three</p>", []string{"3"}},
		{"a token after a newline",
			"<p><span epub:type=\"\npagebreak\" title=\"3\"></span>Three</p>", []string{"3"}},
		{"a token after a carriage return",
			`<p><span epub:type="&#13;pagebreak" title="3"></span>Three</p>`, []string{"3"}},
		{"a page marked by its role",
			`<p><span role="doc-pagebreak" title="6"></span>Six</p>`, []string{"6"}},
		{"a page named by its label",
			`<p><span epub:type="pagebreak" aria-label="7"></span>Seven</p>`, []string{"7"}},
		{"a token that only begins with the word is not it",
			`<p><span epub:type="pagebreaks" title="4"></span>Four</p>`, nil},
		{"a value ending in a space marks nothing",
			`<p><span epub:type="chapter ">Five</span></p>`, nil},
		{"a picture marking a page by its title",
			`<p>One<img epub:type="pagebreak" title="5"/>Two</p>`, []string{"5"}},
		{"a picture marking a page with no title names none",
			`<p><img epub:type="pagebreak"/>Words</p>`, nil},
		{"a page left open at the end of the document",
			`<p>Lead <span epub:type="pagebreak">xiii`, []string{"xiii"}},
		{"a page left open around the whole document",
			`<span epub:type="pagebreak">xiv`, []string{"xiv"}},
		{"a page around the whole document closed before the end",
			`<span epub:type="pagebreak">ix</span> after`, []string{"ix"}},
		{"a page holding an element of its own name",
			`<p><span epub:type="pagebreak"><span>x</span>ii</span></p>`, []string{"xii"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			book, _ := readOne(t, c.markup)
			var got []string
			for _, page := range book.Pages {
				got = append(got, page.Label)
				if page.Offset < 0 || page.Offset > len(book.Text) {
					t.Errorf("page %q is at %d in %d bytes", page.Label, page.Offset, len(book.Text))
				}
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("pages = %q, want %q", got, c.want)
			}
		})
	}

	t.Run("a page stands where its text begins", func(t *testing.T) {
		book, _ := readOne(t, `<p>Before <span epub:type="pagebreak">xii</span> after</p>`)
		if len(book.Pages) != 1 {
			t.Fatalf("pages = %v, want one", book.Pages)
		}
		if want := strings.Index(book.Text, "xii"); book.Pages[0].Offset != want {
			t.Errorf("got page offset %d, want %d", book.Pages[0].Offset, want)
		}
	})
}

// A heading is a part of a book whose navigation names none, at whatever level
// the book wrote it.
func TestHeadingsTheMarkupCarries(t *testing.T) {
	for _, c := range []struct {
		name, markup string
	}{
		{"closed", `<h1>First</h1><p>x</p><h1>Second</h1>`},
		{"the last left open at the end", `<h1>First</h1><p>x</p><h1>Second`},
	} {
		t.Run(c.name, func(t *testing.T) {
			book, _ := readOne(t, c.markup)
			if book.Tier != epub.FromHeadings {
				t.Fatalf("structure = %q, want headings (parts: %v)", book.Tier, book.Parts)
			}
			want := []epub.Part{
				{Title: "First", Offset: 0, Level: 1},
				{Title: "Second", Offset: strings.Index(book.Text, "Second"), Level: 1},
			}
			if !slices.Equal(book.Parts, want) {
				t.Errorf("parts = %v, want %v", book.Parts, want)
			}
		})
	}

	t.Run("each level", func(t *testing.T) {
		book, _ := readOne(t, `<h1>A</h1><h2>B</h2><h3>C</h3><h4>D</h4><h5>E</h5><h6>F</h6>`)
		var want []epub.Part
		for i, title := range []string{"A", "B", "C", "D", "E", "F"} {
			want = append(want, epub.Part{Title: title, Offset: 2 * i, Level: i + 1})
		}
		if !slices.Equal(book.Parts, want) {
			t.Errorf("parts = %v, want %v", book.Parts, want)
		}
	})
}

// An entry of the navigation that carries no name, or points at no document of
// the book, is passed over and the entries after it are read. An entry naming a
// document with no place in it lands where the document begins.
func TestNavigationEntriesLandInTheText(t *testing.T) {
	book := read(t, buildSpineArchive(t, twoDocumentOpf, map[string]string{
		"OEBPS/one.xhtml": `<html><body><p>First words</p></body></html>`,
		"OEBPS/two.xhtml": `<html><body><p>Second words</p><pre>x  y</pre></body></html>`,
		"OEBPS/toc.ncx": `<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
		  <navMap>
		    <navPoint><navLabel><text> </text></navLabel><content src="one.xhtml"/></navPoint>
		    <navPoint><navLabel><text>Gone</text></navLabel><content src="gone.xhtml"/></navPoint>
		    <navPoint><navLabel><text>Here</text></navLabel><content src="#here"/></navPoint>
		    <navPoint><navLabel><text>One</text></navLabel><content src="one.xhtml"/></navPoint>
		    <navPoint><navLabel><text>Two</text></navLabel><content src="two.xhtml"/></navPoint>
		  </navMap>
		  <pageList>
		    <pageTarget><navLabel><text></text></navLabel><content src="one.xhtml"/></pageTarget>
		    <pageTarget value="3"><content src="gone.xhtml"/></pageTarget>
		    <pageTarget value="4"><content src="two.xhtml"/></pageTarget>
		  </pageList>
		</ncx>`,
	}))

	if want := "First words\nSecond words\nx  y\n"; book.Text != want {
		t.Errorf("the text is %q, want %q", book.Text, want)
	}
	second := strings.Index(book.Text, "Second")
	if book.Tier != epub.FromNavigation {
		t.Fatalf("structure = %q, want navigation (parts: %v)", book.Tier, book.Parts)
	}
	if want := []epub.Part{{Title: "One", Offset: 0}, {Title: "Two", Offset: second}}; !slices.Equal(book.Parts, want) {
		t.Errorf("parts = %v, want %v", book.Parts, want)
	}
	if want := []epub.Page{{Label: "4", Offset: second}}; !slices.Equal(book.Pages, want) {
		t.Errorf("pages = %v, want %v", book.Pages, want)
	}
}

const twoDocumentOpf = `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Two</dc:title></metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="a" href="one.xhtml" media-type="application/xhtml+xml"/>
    <item id="b" href="two.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx"><itemref idref="a"/><itemref idref="b"/></spine>
</package>`

// readOne reads a book of one document holding the markup, and that document
// again as the markup a window draws.
func readOne(t *testing.T, markup string) (*epub.Book, *epub.Markup) {
	t.Helper()
	book := read(t, buildSpineArchive(t, oneDocumentOpf, map[string]string{"OEBPS/one.xhtml": markup}))
	rendered, err := book.Markup("OEBPS/one.xhtml")
	if err != nil {
		t.Fatalf("markup: %v", err)
	}
	return book, rendered
}

// A pre left open at the end of a document does not keep the next document
// preformatted.
func TestPreLeftOpenEndsWithItsDocument(t *testing.T) {
	book := read(t, buildSpineArchive(t, twoDocumentOpf, map[string]string{
		"OEBPS/one.xhtml": `<pre>kept  as it was`,
		"OEBPS/two.xhtml": `<p>one   run</p>`,
		"OEBPS/toc.ncx":   `<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1"><navMap></navMap></ncx>`,
	}))
	if want := "kept  as it was\none run\n"; book.Text != want {
		t.Errorf("the text is %q, want %q", book.Text, want)
	}
}

// A line break contains no text, and the content after it is a sibling node.
func TestLineBreakContainsNoText(t *testing.T) {
	_, markup := readOne(t, `<p>a<br/>b</p>`)
	if len(markup.Nodes) != 1 || markup.Nodes[0].Name != "p" {
		t.Fatalf("nodes = %+v, want one paragraph", markup.Nodes)
	}
	var names []string
	for _, child := range markup.Nodes[0].Children {
		names = append(names, child.Name)
		if child.Name == "br" && len(child.Children) != 0 {
			t.Errorf("the break holds %+v, want nothing", child.Children)
		}
	}
	if got := slices.Index(names, "br"); got < 0 {
		t.Errorf("the paragraph holds %q, want a break among them", names)
	}
}

// Page breaks declared inside head or title metadata elements are ignored.
func TestPagebreakInHeadIgnored(t *testing.T) {
	book, _ := readOne(t, `<head><title epub:type="pagebreak" title="1">T</title><span epub:type="pagebreak" title="2"></span></head><body><p>Text</p></body>`)
	if len(book.Pages) != 0 {
		t.Errorf("got pages %v, want none", book.Pages)
	}
}
