package epub_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/plugins/readers.epub/backend/epub"
)

// The window a book is read in draws elements, not a stream of text, and it
// takes every offset it needs off the markup it was handed.
func TestMarkup(t *testing.T) {
	book := read(t, tinyBook(t, map[string]string{secondDoc: "variant/drawn.xhtml"}))
	rendered, err := book.Markup(secondDoc)
	if err != nil {
		t.Fatalf("markup: %v", err)
	}

	t.Run("the words are the words of the text", func(t *testing.T) {
		doc := document(t, book, secondDoc)
		want := book.Text[doc.Offset : doc.Offset+doc.Length]
		if got := getNodeText(rendered.Nodes); got != want {
			t.Fatalf("the markup says\n%q\nand the text is\n%q", got, want)
		}
	})

	t.Run("every run stands where it says", func(t *testing.T) {
		for _, run := range getTextNodes(rendered.Nodes) {
			if !strings.HasPrefix(book.Text[run.Offset:], run.Text) {
				t.Errorf("%q says it begins at %d, where the text is %q",
					run.Text, run.Offset, excerpt(book.Text, run.Offset))
			}
		}
		if quoted := element(t, rendered.Nodes, "blockquote"); !strings.Contains(getNodeText(quoted.Children), "Eta") {
			t.Error("the quotation is not a quotation")
		}
		if em := element(t, rendered.Nodes, "em"); getNodeText(em.Children) != "emphatic" {
			t.Errorf("the emphasis says %q", getNodeText(em.Children))
		}
	})

	t.Run("the book's own styling does not survive", func(t *testing.T) {
		for _, node := range getEveryNode(rendered.Nodes) {
			for _, a := range node.Attributes {
				if a.Name == "class" || a.Name == "style" {
					t.Errorf("%s carries %s=%q", node.Name, a.Name, a.Value)
				}
			}
		}
		if head := element(t, rendered.Nodes, "h4"); attribute(head, "id") != "three" {
			t.Errorf("the heading lost its id, which is what a link inside the book lands on")
		}
	})

	t.Run("what carries no words of the book is gone", func(t *testing.T) {
		for _, name := range []string{"script", "style", "link", "head", "title", "form"} {
			if found := find(rendered.Nodes, name); found != nil {
				t.Errorf("a %s element was drawn", name)
			}
		}
		if strings.Contains(getNodeText(rendered.Nodes), "forbidden") {
			t.Error("a script or a stylesheet reached the markup")
		}
		// A form is not drawn and the words inside it are: the text carries
		// them, and the two say the same thing.
		if !strings.Contains(getNodeText(rendered.Nodes), "Iota") {
			t.Error("the words inside the form are missing")
		}
	})

	t.Run("an address inside the book names an entry of the archive", func(t *testing.T) {
		links := findAll(rendered.Nodes, "a")
		if len(links) != 5 {
			t.Fatalf("links = %d, want the five the document holds", len(links))
		}
		if got := attribute(links[0], "href"); got != "OEBPS/first.xhtml#alpha" {
			t.Errorf("the link inside the book points at %q", got)
		}
		// An address leading out of the book keeps its scheme.
		if got := attribute(links[1], "href"); got != "https://example.org/away" {
			t.Errorf("the link out of the book points at %q", got)
		}
		if got := attribute(element(t, rendered.Nodes, "img"), "src"); got != "OEBPS/pictures/plate.png" {
			t.Errorf("the picture is at %q, want the entry of the archive", got)
		}
		if got := attribute(element(t, rendered.Nodes, "img"), "alt"); got != "A plate" {
			t.Errorf("the picture is described as %q", got)
		}
	})

	// The window a book is read in is a webview, so an address that is a program
	// leads nowhere, and one that hides its scheme behind a line ending is that
	// address and not a file of the book.
	t.Run("an address that runs a program leads nowhere", func(t *testing.T) {
		for _, link := range findAll(rendered.Nodes, "a")[2:] {
			if got := attribute(link, "href"); got != "" {
				t.Errorf("a link points at %q", got)
			}
		}
		// A book draws the files it carries, and asks nobody for a picture: a
		// picture fetched is a book reporting that it was opened.
		pictures := findAll(rendered.Nodes, "img")
		if len(pictures) != 2 {
			t.Fatalf("pictures = %d, want the two the document holds", len(pictures))
		}
		if got := attribute(pictures[1], "src"); got != "" {
			t.Errorf("a picture off the machine is drawn from %q", got)
		}
		if got := attribute(pictures[1], "alt"); got == "" {
			t.Error("the picture lost what it is described as")
		}
	})

	t.Run("a document the book was not read from", func(t *testing.T) {
		if _, err := book.Markup("OEBPS/gone.xhtml"); !errors.Is(err, epub.ErrNoDocument) {
			t.Errorf("err = %v, want %v", err, epub.ErrNoDocument)
		}
	})
}

// A spine document that is an SVG is a cover drawn around a picture, and there
// SVG served from the window's own origin is a document there, and its own
// addresses reach no entry of the archive.
func TestASpineDocumentThatIsACoverGeneratedAroundAPicture(t *testing.T) {
	book := read(t, coverBook(t, `<svg xmlns="http://www.w3.org/2000/svg"
		xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 600 800">
		<image width="600" height="800" xlink:href="pictures/plate.png"/></svg>`))

	rendered, err := book.Markup("OEBPS/cover.svg")
	if err != nil {
		t.Fatalf("markup: %v", err)
	}
	if len(rendered.Nodes) == 0 || rendered.Nodes[0].Name != "img" {
		t.Fatalf("the cover was drawn as %v", rendered.Nodes)
	}
	if got := attribute(&rendered.Nodes[0], "src"); got != "OEBPS/pictures/plate.png" {
		t.Errorf("the cover is drawn from %q", got)
	}
	if _, err := book.Entry(attribute(&rendered.Nodes[0], "src")); err != nil {
		t.Errorf("the cover is drawn from an address the archive answers with %v", err)
	}
	doc := document(t, book, "OEBPS/cover.svg")
	if want := book.Text[doc.Offset : doc.Offset+doc.Length]; getNodeText(rendered.Nodes) != want {
		t.Errorf("the picture says %q and its text is %q", getNodeText(rendered.Nodes), want)
	}
}

// A cover naming no picture of the archive draws none: an address the window
// refuses is a broken picture on the page.
func TestACoverThatWrapsNoPictureDrawsNone(t *testing.T) {
	for _, one := range []struct{ name, svg string }{
		{"nothing at all", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 600 800">
			<rect width="600" height="800"/></svg>`},
		{"an entry the archive does not hold", `<svg xmlns="http://www.w3.org/2000/svg"
			xmlns:xlink="http://www.w3.org/1999/xlink"><image xlink:href="gone.png"/></svg>`},
		{"an address off the machine", `<svg xmlns="http://www.w3.org/2000/svg"
			xmlns:xlink="http://www.w3.org/1999/xlink"><image xlink:href="https://example.invalid/p.png"/></svg>`},
		{"a picture another element names", `<svg xmlns="http://www.w3.org/2000/svg"><use href="pictures/plate.png"/></svg>`},
		{"a picture an image names by another attribute", `<svg xmlns="http://www.w3.org/2000/svg">
			<image title="pictures/plate.png"/></svg>`},
	} {
		t.Run(one.name, func(t *testing.T) {
			rendered, err := read(t, coverBook(t, one.svg)).Markup("OEBPS/cover.svg")
			if err != nil {
				t.Fatalf("markup: %v", err)
			}
			if found := findAll(rendered.Nodes, "img"); len(found) != 0 {
				t.Errorf("the cover drew %d pictures", len(found))
			}
		})
	}
}

// A cover written in the version of SVG that names what it draws with href, and
// one carrying the picture itself.
func TestACoverNamesThePictureItDraws(t *testing.T) {
	const content = "data:image/png;base64,iVBORw0KGgo="
	for _, one := range []struct{ svg, want string }{
		{`<svg xmlns="http://www.w3.org/2000/svg"><image href="pictures/plate.png"/></svg>`,
			"OEBPS/pictures/plate.png"},
		{`<svg xmlns="http://www.w3.org/2000/svg"><g><image href="` + content + `"/></g></svg>`, content},
		// An entity the SVG does not declare and an encoding other than UTF-8
		// still draw the picture.
		{`<svg xmlns="http://www.w3.org/2000/svg"><title>Plate&nbsp;one</title>
			<image href="pictures/plate.png"/></svg>`, "OEBPS/pictures/plate.png"},
		{`<?xml version="1.0" encoding="iso-8859-1"?>
			<svg xmlns="http://www.w3.org/2000/svg"><image href="pictures/plate.png"/></svg>`,
			"OEBPS/pictures/plate.png"},
	} {
		rendered, err := read(t, coverBook(t, one.svg)).Markup("OEBPS/cover.svg")
		if err != nil {
			t.Fatalf("markup: %v", err)
		}
		found := findAll(rendered.Nodes, "img")
		if len(found) != 1 {
			t.Fatalf("the cover drew %d pictures", len(found))
		}
		if got := attribute(found[0], "src"); got != one.want {
			t.Errorf("the cover is drawn from %q, want %q", got, one.want)
		}
	}
}

// coverBook is a book whose spine opens with an SVG cover.
func coverBook(t *testing.T, cover string) []byte {
	t.Helper()
	return buildSpineArchive(t, `<?xml version="1.0"?>
		<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
		  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>A Cover</dc:title></metadata>
		  <manifest>
		    <item id="a" href="cover.svg" media-type="image/svg+xml"/>
		    <item id="b" href="one.xhtml" media-type="application/xhtml+xml"/>
		    <item id="c" href="pictures/plate.png" media-type="image/png"/>
		  </manifest>
		  <spine><itemref idref="a"/><itemref idref="b"/></spine>
		</package>`, map[string]string{
		"OEBPS/cover.svg":          cover,
		"OEBPS/one.xhtml":          `<html><body><p>Nu.</p></body></html>`,
		"OEBPS/pictures/plate.png": "\x89PNG\r\n\x1a\n",
	})
}

// document is one document of the book, by name.
func document(t *testing.T, book *epub.Book, path string) epub.Document {
	t.Helper()
	for _, doc := range book.Documents {
		if doc.Path == path {
			return doc
		}
	}
	t.Fatalf("%s is not a document of the book", path)
	return epub.Document{}
}

// element is the first element of a name, and a failure when there is none.
func element(t *testing.T, nodes []epub.Node, name string) *epub.Node {
	t.Helper()
	found := find(nodes, name)
	if found == nil {
		t.Fatalf("no %s element was drawn", name)
	}
	return found
}

func find(nodes []epub.Node, name string) *epub.Node {
	if found := findAll(nodes, name); len(found) > 0 {
		return found[0]
	}
	return nil
}

func findAll(nodes []epub.Node, name string) []*epub.Node {
	var out []*epub.Node
	for _, node := range getEveryNode(nodes) {
		if node.Name == name {
			out = append(out, node)
		}
	}
	return out
}

// getEveryNode node of a document, in reading order.
func getEveryNode(nodes []epub.Node) []*epub.Node {
	var out []*epub.Node
	for i := range nodes {
		out = append(out, &nodes[i])
		out = append(out, getEveryNode(nodes[i].Children)...)
	}
	return out
}

// getTextNodes answers the nodes that are text.
func getTextNodes(nodes []epub.Node) []*epub.Node {
	var out []*epub.Node
	for _, node := range getEveryNode(nodes) {
		if node.Name == "" {
			out = append(out, node)
		}
	}
	return out
}

func attribute(node *epub.Node, name string) string {
	for _, a := range node.Attributes {
		if a.Name == name {
			return a.Value
		}
	}
	return ""
}

// A link keeps an address in a scheme a person may be sent to, or an entry of
// the archive. A scheme is a letter followed by letters, digits, plus, minus and
// dots, and any other scheme leads nowhere.
func TestWhereALinkLeads(t *testing.T) {
	for _, c := range []struct{ href, want string }{
		{" mailto:someone@example.org", "mailto:someone@example.org"},
		{"x:payload", ""},
		{"AZaz09+-.:payload", ""},
		{"a1:payload", ""},
		{"1ab:payload", "OEBPS/1ab:payload"},
		{"~ab:payload", "OEBPS/~ab:payload"},
		{"a_b:payload", "OEBPS/a_b:payload"},
		{"notes/one:two.xhtml", "OEBPS/notes/one:two.xhtml"},
		{"#here", "#here"},
		{"", ""},
	} {
		t.Run(c.href, func(t *testing.T) {
			_, markup := readOne(t, `<p><a href="`+c.href+`">x</a></p>`)
			if got := attribute(element(t, markup.Nodes, "a"), "href"); got != c.want {
				t.Errorf("the link points at %q, want %q", got, c.want)
			}
		})
	}
}

// A picture is a file the book carries or one written into the markup as base64.
func TestWhatAPictureIsGeneratedFrom(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"data:image/png;base64,iVBORw0KGgo=", "data:image/png;base64,iVBORw0KGgo="},
		{"data:text/plain;base64,QQ==", ""},
		{"data:image/svg+xml,%3Csvg%2F%3E", ""},
	} {
		t.Run(c.src, func(t *testing.T) {
			_, markup := readOne(t, `<p><img src="`+c.src+`" alt="A"/></p>`)
			if got := attribute(element(t, markup.Nodes, "img"), "src"); got != c.want {
				t.Errorf("the picture is drawn from %q, want %q", got, c.want)
			}
		})
	}
}

// An address that leads nowhere goes, and the attributes after it stay.
func TestAnAddressThatLeadsNowhereConsumesNothingElse(t *testing.T) {
	_, markup := readOne(t, `<p><a href="javascript:void(0)" id="k">x</a>`+
		`<img src="https://example.org/p.png" alt="A plate"/></p>`)
	link := element(t, markup.Nodes, "a")
	if want := []epub.Attribute{{Name: "id", Value: "k"}}; !slices.Equal(link.Attributes, want) {
		t.Errorf("the link carries %v, want %v", link.Attributes, want)
	}
	picture := element(t, markup.Nodes, "img")
	if want := []epub.Attribute{{Name: "alt", Value: "A plate"}}; !slices.Equal(picture.Attributes, want) {
		t.Errorf("the picture carries %v, want %v", picture.Attributes, want)
	}
}

// An element standing on the space a line's end takes back stands where the
// line ends.
func TestAnElementOnASpaceTakenBackExistsAtTheLineEnd(t *testing.T) {
	book, markup := readOne(t, `<p>A <b></b></p>`)
	if book.Text != "A\n" {
		t.Fatalf("the text is %q", book.Text)
	}
	if got := element(t, markup.Nodes, "b").Offset; got != 1 {
		t.Errorf("got element offset %d, want 1", got)
	}
	if got := getNodeText(markup.Nodes); got != book.Text {
		t.Errorf("the markup says %q, want %q", got, book.Text)
	}
}

// itself when no cell follows. A run written after a table is not in it.
func TestRunsInATable(t *testing.T) {
	for _, c := range []struct {
		name, markup, in, want string
	}{
		{"a cell holds the end of its row's line", `<table><tr><td>A</td></tr></table>`, "td", "A\n"},
		{"text after a table is not in its last cell",
			`<div><table><tr><td>A</td></tr></table>B</div>`, "td", "A\n"},
		{"a run in a row before its cell is the cell's",
			`<table><tr><template>X</template><td>A</td></tr></table>`, "td", "X A\n"},
		{"a run in a table before its rows is the first cell's",
			`<table><template>X</template><tr><td>A</td></tr></table>`, "td", "X\nA\n"},
		{"a run in a table that opens no cell is the table's",
			`<table><template>X</template></table>`, "table", "X\n"},
		{"a run in a row that opens no cell is the table's, not the row's",
			`<table><tr><template>X</template></tr></table>`, "tr", ""},
		{"a run waiting for a cell is no caption's",
			`<table><template>X</template><caption><b>C</b></caption></table>`, "b", "C"},
		{"a table whose text is all in an element keeps its line end",
			`<table><template><b>X</b></template></table>`, "table", "X\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			book, markup := readOne(t, c.markup)
			if got := getNodeText(markup.Nodes); got != book.Text {
				t.Fatalf("the markup says %q and the text is %q", got, book.Text)
			}
			holder := element(t, markup.Nodes, c.in)
			if got := getNodeText(holder.Children); got != c.want {
				t.Errorf("the %s says %q, want %q", c.in, got, c.want)
			}
			if len(holder.Children) > 0 && holder.Offset != holder.Children[0].Offset {
				t.Errorf("%s is at %d and content at %d",
					c.in, holder.Offset, holder.Children[0].Offset)
			}
			if c.in == "td" {
				for _, target := range element(t, markup.Nodes, "table").Children {
					if target.Name == "" {
						t.Errorf("the table holds %q outside its cell", target.Text)
					}
				}
			}
		})
	}
}

// What the book holds beside its spine is not a document of it.
func TestAnEntryAdjacentTheSpineIsNoDocument(t *testing.T) {
	book := read(t, coverBook(t, `<svg xmlns="http://www.w3.org/2000/svg"/>`))
	if _, err := book.Markup("OEBPS/pictures/plate.png"); !errors.Is(err, epub.ErrNoDocument) {
		t.Errorf("err = %v, want %v", err, epub.ErrNoDocument)
	}
}
