package epub

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// An extractor builds the one text of a book and remembers the offsets of the
// places the markup marks.
type extractor struct {
	out []byte

	// path is the document being read, and starts holds where each document
	// began, so that an href with no fragment still has an offset.
	path   string
	starts map[string]int

	// anchors maps "document#id" to the offset the element with that id begins
	// at. This is how a navigation entry that points inside a document lands.
	anchors map[string]int

	headings   []Part
	pagebreaks []Page

	// markup gathers the elements of the document being read, and is nil when
	// only the text is wanted.
	markup *builder

	// preformattedDepth counts the pre elements open around the text being written.
	preformattedDepth int
}

func newExtractor() *extractor {
	return &extractor{
		starts:  map[string]int{},
		anchors: map[string]int{},
	}
}

// document reads one spine document into the text.
//
// The markup is parsed as HTML: a document with three opening body tags is still
// a document a person can read.
func (x *extractor) document(docPath string, markup []byte) Document {
	x.breakLine()
	start := len(x.out)
	x.path = docPath
	x.starts[docPath] = start

	if root, err := html.Parse(bytes.NewReader(markup)); err == nil {
		x.node(root)
	}
	x.breakLine()
	return Document{Path: docPath, Offset: start, Length: len(x.out) - start}
}

func (x *extractor) node(n *html.Node) {
	if n == nil {
		return
	}
	switch n.Type {
	case html.TextNode:
		x.write(n.Data)
		return
	case html.ElementNode:
	case html.DocumentNode:
		x.children(n)
		return
	default:
		return
	}

	switch n.DataAtom {
	case atom.Head, atom.Script, atom.Style, atom.Title:
		// Not text of the book.
		return
	case atom.Br:
		x.breakLine()
		// The line it ended is behind it, so the break stands at the line it
		// begins.
		if x.markup.opened(n, len(x.out)) {
			x.markup.close()
		}
		return
	case atom.Td, atom.Th:
		x.write(" ")
	}

	block := blocks[n.DataAtom]
	if block {
		x.breakLine()
	}
	if id := attribute(n, "id"); id != "" {
		x.anchors[x.path+"#"+id] = len(x.out)
	}
	start := len(x.out)

	if n.DataAtom == atom.Pre {
		x.preformattedDepth++
	}
	element := x.markup.opened(n, start)
	x.children(n)
	if n.DataAtom == atom.Pre {
		x.preformattedDepth--
	}

	if level := headingLevel(n.DataAtom); level > 0 {
		if title := tidy(string(x.out[start:])); title != "" {
			x.headings = append(x.headings, Part{Title: title, Offset: start, Level: level})
		}
	}
	if label, ok := printedPage(n); ok {
		if label == "" {
			label = tidy(string(x.out[start:]))
		}
		if label != "" {
			x.pagebreaks = append(x.pagebreaks, Page{Label: label, Offset: start})
		}
	}
	if block {
		x.breakLine()
	}
	// The line a block ends is the block's own, so it closes over it.
	if element {
		x.markup.close()
	}
}

func (x *extractor) children(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		x.node(c)
	}
}

// write appends text. A run of space is one space, and text inside a pre element
// is kept as it was written.
func (x *extractor) write(text string) {
	start := len(x.out)
	if x.preformattedDepth > 0 {
		x.out = append(x.out, text...)
	} else {
		for _, r := range text {
			if unicode.IsSpace(r) {
				if last, ok := x.last(); ok && last != ' ' && last != '\n' {
					x.out = append(x.out, ' ')
				}
				continue
			}
			x.out = utf8.AppendRune(x.out, r)
		}
	}
	if len(x.out) > start {
		x.markup.run(start)
	}
}

// breakLine ends the line one block of text sits on.
func (x *extractor) breakLine() {
	if last, ok := x.last(); ok && last == ' ' {
		x.out = x.out[:len(x.out)-1]
	}
	if last, ok := x.last(); ok && last != '\n' {
		start := len(x.out)
		x.out = append(x.out, '\n')
		x.markup.run(start)
	}
}

func (x *extractor) last() (byte, bool) {
	if len(x.out) == 0 {
		return 0, false
	}
	return x.out[len(x.out)-1], true
}

// named turns navigation entries into parts. An entry whose document is not in
// the book, or which carries no name, is not one.
func (x *extractor) named(entries []entry) []Part {
	var out []Part
	for _, e := range entries {
		title := tidy(e.title)
		if title == "" {
			continue
		}
		offset, ok := x.offsetOf(e.href)
		if !ok {
			continue
		}
		out = append(out, Part{Title: title, Offset: offset})
	}
	return out
}

// pagesAt turns the entries of a page list into pages. A book with no page list
// keeps the page breaks its markup marks.
func (x *extractor) pagesAt(entries []entry) []Page {
	var out []Page
	for _, e := range entries {
		label := tidy(e.title)
		if label == "" {
			continue
		}
		offset, ok := x.offsetOf(e.href)
		if !ok {
			continue
		}
		out = append(out, Page{Label: label, Offset: offset})
	}
	if len(out) == 0 {
		return x.pagebreaks
	}
	return out
}

// offsetOf is where an href lands in the text: the place its fragment marks, or
// the start of the document it names.
func (x *extractor) offsetOf(href string) (int, bool) {
	target, fragment := splitHref(href)
	if target == "" {
		return 0, false
	}
	if fragment != "" {
		if offset, ok := x.anchors[target+"#"+fragment]; ok {
			return offset, true
		}
	}
	offset, ok := x.starts[target]
	return offset, ok
}

// attribute reads an attribute of the element by name. The markup is read as
// HTML, where a prefix is part of the name: `epub:type` is an attribute called
// "epub:type".
func attribute(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
		}
	}
	return ""
}

// printedPage reports whether an element marks a page of the printed book, and
// the number it gives.
func printedPage(n *html.Node) (label string, marks bool) {
	marks = hasToken(attribute(n, "epub:type"), "pagebreak") ||
		hasToken(attribute(n, "role"), "doc-pagebreak")
	if !marks {
		return "", false
	}
	for _, name := range []string{"title", "aria-label"} {
		if label := tidy(attribute(n, name)); label != "" {
			return label, true
		}
	}
	return "", true
}

// headingLevel is the level of a heading element, and 0 for anything else. The
// level a book chose carries no meaning of its own: a book whose sections are
// h4 has sections.
func headingLevel(a atom.Atom) int {
	switch a {
	case atom.H1:
		return 1
	case atom.H2:
		return 2
	case atom.H3:
		return 3
	case atom.H4:
		return 4
	case atom.H5:
		return 5
	case atom.H6:
		return 6
	}
	return 0
}

// blocks are the elements whose text stands on a line of its own.
var blocks = map[atom.Atom]bool{
	atom.Address:    true,
	atom.Article:    true,
	atom.Aside:      true,
	atom.Blockquote: true,
	atom.Body:       true,
	atom.Caption:    true,
	atom.Dd:         true,
	atom.Div:        true,
	atom.Dl:         true,
	atom.Dt:         true,
	atom.Figcaption: true,
	atom.Figure:     true,
	atom.Footer:     true,
	atom.H1:         true,
	atom.H2:         true,
	atom.H3:         true,
	atom.H4:         true,
	atom.H5:         true,
	atom.H6:         true,
	atom.Header:     true,
	atom.Hr:         true,
	atom.Li:         true,
	atom.Main:       true,
	atom.Nav:        true,
	atom.Ol:         true,
	atom.P:          true,
	atom.Pre:        true,
	atom.Section:    true,
	atom.Table:      true,
	atom.Tr:         true,
	atom.Ul:         true,
}
