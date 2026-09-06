package epub_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/epub"
)

// The window a book is read in draws elements, not a stream of text, and it
// takes every offset it needs off the markup it was handed.
func TestMarkup(t *testing.T) {
	book := read(t, tinyBook(t, map[string]string{secondDoc: "variant/drawn.xhtml"}))
	drawn, err := book.Markup(secondDoc)
	if err != nil {
		t.Fatalf("markup: %v", err)
	}

	t.Run("the words are the words of the text", func(t *testing.T) {
		doc := document(t, book, secondDoc)
		want := book.Text[doc.Offset : doc.Offset+doc.Length]
		if got := said(drawn.Nodes); got != want {
			t.Fatalf("the markup says\n%q\nand the text is\n%q", got, want)
		}
	})

	t.Run("every run stands where it says", func(t *testing.T) {
		for _, run := range runs(drawn.Nodes) {
			if !strings.HasPrefix(book.Text[run.Offset:], run.Text) {
				t.Errorf("%q says it begins at %d, where the text is %q",
					run.Text, run.Offset, excerpt(book.Text, run.Offset))
			}
		}
		if quoted := element(t, drawn.Nodes, "blockquote"); !strings.Contains(said(quoted.Children), "Eta") {
			t.Error("the quotation is not a quotation")
		}
		if em := element(t, drawn.Nodes, "em"); said(em.Children) != "emphatic" {
			t.Errorf("the emphasis says %q", said(em.Children))
		}
	})

	t.Run("the book's own styling does not survive", func(t *testing.T) {
		for _, node := range every(drawn.Nodes) {
			for _, a := range node.Attributes {
				if a.Name == "class" || a.Name == "style" {
					t.Errorf("%s carries %s=%q", node.Name, a.Name, a.Value)
				}
			}
		}
		if head := element(t, drawn.Nodes, "h4"); attribute(head, "id") != "three" {
			t.Errorf("the heading lost its id, which is what a link inside the book lands on")
		}
	})

	t.Run("what carries no words of the book is gone", func(t *testing.T) {
		for _, name := range []string{"script", "style", "link", "head", "title", "form"} {
			if found := find(drawn.Nodes, name); found != nil {
				t.Errorf("a %s element was drawn", name)
			}
		}
		if strings.Contains(said(drawn.Nodes), "forbidden") {
			t.Error("a script or a stylesheet reached the markup")
		}
		// A form is not drawn and the words inside it are: the text carries
		// them, and the two say the same thing.
		if !strings.Contains(said(drawn.Nodes), "Iota") {
			t.Error("the words inside the form are missing")
		}
	})

	t.Run("an address inside the book names an entry of the archive", func(t *testing.T) {
		links := findAll(drawn.Nodes, "a")
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
		if got := attribute(element(t, drawn.Nodes, "img"), "src"); got != "OEBPS/pictures/plate.png" {
			t.Errorf("the picture is at %q, want the entry of the archive", got)
		}
		if got := attribute(element(t, drawn.Nodes, "img"), "alt"); got != "A plate" {
			t.Errorf("the picture is described as %q", got)
		}
	})

	// The window a book is read in is a webview, so an address that is a program
	// leads nowhere, and one that hides its scheme behind a line ending is that
	// address and not a file of the book.
	t.Run("an address that runs a program leads nowhere", func(t *testing.T) {
		for _, link := range findAll(drawn.Nodes, "a")[2:] {
			if got := attribute(link, "href"); got != "" {
				t.Errorf("a link points at %q", got)
			}
		}
		// A book draws the files it carries, and asks nobody for a picture: a
		// picture fetched is a book reporting that it was opened.
		pictures := findAll(drawn.Nodes, "img")
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

// A spine document that is a picture is a cover, and there is nothing in it to
// reflow. It is answered as the picture it is, and what a reader draws is the
// entry of the archive.
func TestASpineDocumentThatIsAPicture(t *testing.T) {
	book := read(t, spined(t, `<?xml version="1.0"?>
		<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
		  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>A Cover</dc:title></metadata>
		  <manifest>
		    <item id="a" href="cover.svg" media-type="image/svg+xml"/>
		    <item id="b" href="one.xhtml" media-type="application/xhtml+xml"/>
		  </manifest>
		  <spine><itemref idref="a"/><itemref idref="b"/></spine>
		</package>`, map[string]string{
		"OEBPS/cover.svg": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 600 800">
			<image width="600" height="800" xlink:href="plate.png"/></svg>`,
		"OEBPS/one.xhtml": `<html><body><p>Nu.</p></body></html>`,
	}))

	drawn, err := book.Markup("OEBPS/cover.svg")
	if err != nil {
		t.Fatalf("markup: %v", err)
	}
	if len(drawn.Nodes) == 0 || drawn.Nodes[0].Name != "img" {
		t.Fatalf("the cover was drawn as %v", drawn.Nodes)
	}
	if got := attribute(&drawn.Nodes[0], "src"); got != "OEBPS/cover.svg" {
		t.Errorf("the cover is at %q", got)
	}
	doc := document(t, book, "OEBPS/cover.svg")
	if want := book.Text[doc.Offset : doc.Offset+doc.Length]; said(drawn.Nodes) != want {
		t.Errorf("the picture says %q and its text is %q", said(drawn.Nodes), want)
	}
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
	for _, node := range every(nodes) {
		if node.Name == name {
			out = append(out, node)
		}
	}
	return out
}

// every node of a document, in reading order.
func every(nodes []epub.Node) []*epub.Node {
	var out []*epub.Node
	for i := range nodes {
		out = append(out, &nodes[i])
		out = append(out, every(nodes[i].Children)...)
	}
	return out
}

// runs are the nodes that are text.
func runs(nodes []epub.Node) []*epub.Node {
	var out []*epub.Node
	for _, node := range every(nodes) {
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
