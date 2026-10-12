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
	start := len(x.out)
	x.path = docPath
	x.starts[docPath] = start

	if x.markup != nil {
		if root, err := html.Parse(bytes.NewReader(markup)); err == nil {
			x.children(root)
		}
	} else {
		x.tokenize(markup)
	}
	x.breakLine()
	return Document{Path: docPath, Offset: start, Length: len(x.out) - start}
}

func (x *extractor) node(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		x.write(n.Data)
		return
	case html.ElementNode:
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
		if x.markup.openElement(n, len(x.out)) {
			x.markup.close()
		}
		return
	}

	block := blocks[n.DataAtom]
	if block {
		x.breakLine()
	}

	if n.DataAtom == atom.Pre {
		x.preformattedDepth++
	}
	element := x.markup.openElement(n, len(x.out))
	if n.DataAtom == atom.Td || n.DataAtom == atom.Th {
		// A cell is set off from the one before it by a space, and the space
		// stands inside the cell it opens.
		x.write(" ")
	}
	x.children(n)
	if n.DataAtom == atom.Pre {
		x.preformattedDepth--
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

func (x *extractor) writeBytes(b []byte) {
	if x.preformattedDepth > 0 {
		x.out = append(x.out, b...)
		return
	}
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if unicode.IsSpace(r) {
			if last, ok := x.last(); ok && last != ' ' && last != '\n' {
				x.out = append(x.out, ' ')
			}
		} else {
			x.out = append(x.out, b[i:i+size]...)
		}
		i += size
	}
}

// A frame tracks an open element during tokenization.
type frame struct {
	atom        atom.Atom
	start       int
	block       bool
	heading     int
	isPagebreak bool
	pageLabel   string
}

// tokenize reads markup into text using a streaming HTML tokenizer.
func (x *extractor) tokenize(markup []byte) {
	z := html.NewTokenizer(bytes.NewReader(markup))
	var stackBuf [32]frame
	stack := stackBuf[:0]
	var skipAtom atom.Atom
	dropLeadingNewline := false

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			for i := len(stack) - 1; i >= 0; i-- {
				x.closeFrame(stack[i])
			}
			x.preformattedDepth = 0
			return

		case html.CommentToken, html.DoctypeToken:
		case html.TextToken:
			if skipAtom == 0 {
				text := z.Text()
				if dropLeadingNewline {
					text = bytes.TrimPrefix(bytes.TrimPrefix(text, []byte{'\r'}), []byte{'\n'})
				}
				x.writeBytes(text)
			}
			dropLeadingNewline = false

		case html.StartTagToken, html.SelfClosingTagToken:
			dropLeadingNewline = false
			name, hasAttr := z.TagName()
			a := atom.Lookup(name)

			if skipAtom != 0 {
				if a == atom.Body || blocks[a] {
					skipAtom = 0
				} else {
					// gomutants:disable-next-line INVERT_LOOP_CTRL reason="break leaves the switch and ends the iteration like continue"
					continue
				}
			}

			if a == atom.Head || a == atom.Script || a == atom.Style || a == atom.Title {
				if tt == html.StartTagToken {
					skipAtom = a
				}
				// gomutants:disable-next-line INVERT_LOOP_CTRL reason="break leaves the switch and ends the iteration like continue"
				continue
			}

			if a == atom.Br {
				x.breakLine()
				// gomutants:disable-next-line INVERT_LOOP_CTRL reason="break leaves the switch and ends the iteration like continue"
				continue
			}

			block := blocks[a]
			if block {
				x.breakLine()
			}

			var id string
			var isPagebreak bool
			var pageLabel string
			var titleAttr string
			var ariaLabelAttr string

			for hasAttr {
				var key, val []byte
				key, val, hasAttr = z.TagAttr()
				switch {
				case bytes.Equal(key, []byte("id")):
					id = string(val)
				case bytes.Equal(key, []byte("epub:type")):
					if hasTokenBytes(val, "pagebreak") {
						isPagebreak = true
					}
				case bytes.Equal(key, []byte("role")):
					if hasTokenBytes(val, "doc-pagebreak") {
						isPagebreak = true
					}
				case bytes.Equal(key, []byte("title")):
					titleAttr = string(val)
				case bytes.Equal(key, []byte("aria-label")):
					ariaLabelAttr = string(val)
				}
			}

			if id != "" {
				x.anchors[x.path+"#"+id] = len(x.out)
			}
			if isPagebreak {
				if t := normaliseSpaces(titleAttr); t != "" {
					pageLabel = t
				} else if al := normaliseSpaces(ariaLabelAttr); al != "" {
					pageLabel = al
				}
			}

			start := len(x.out)

			if a == atom.Pre {
				x.preformattedDepth++
				dropLeadingNewline = true
			}
			if a == atom.Listing || a == atom.Textarea {
				dropLeadingNewline = true
			}
			if a == atom.Td || a == atom.Th {
				x.writeBytes([]byte(" "))
			}

			level := headingLevel(a)

			if isVoidTag(a) {
				x.closeFrame(frame{start: start, isPagebreak: isPagebreak, pageLabel: pageLabel})
				if block {
					x.breakLine()
				}
			} else {
				stack = append(stack, frame{
					atom:        a,
					start:       start,
					block:       block,
					heading:     level,
					isPagebreak: isPagebreak,
					pageLabel:   pageLabel,
				})
			}

		case html.EndTagToken:
			name, _ := z.TagName()
			a := atom.Lookup(name)

			if skipAtom != 0 {
				if a == skipAtom {
					skipAtom = 0
				}
				// gomutants:disable-next-line INVERT_LOOP_CTRL reason="break leaves the switch and ends the iteration like continue"
				continue
			}

			if a == atom.Br || a == atom.Body || a == atom.Html {
				if a == atom.Br {
					x.breakLine()
				}
				// gomutants:disable-next-line INVERT_LOOP_CTRL reason="break leaves the switch and ends the iteration like continue"
				continue
			}

			found := -1
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].atom == a {
					found = i
					break
				}
			}

			if found >= 0 {
				for i := len(stack) - 1; i >= found; i-- {
					x.closeFrame(stack[i])
					if stack[i].block {
						x.breakLine()
					}
				}
				stack = stack[:found]
			} else if blocks[a] {
				x.breakLine()
			}
		}
	}
}

// closeFrame records the heading and the page break an element that has ended
// carried, and leaves a pre element.
func (x *extractor) closeFrame(f frame) {
	if f.atom == atom.Pre {
		x.preformattedDepth--
	}
	if f.heading > 0 {
		if title := normaliseSpaces(string(x.out[f.start:])); title != "" {
			x.headings = append(x.headings, Part{Title: title, Offset: f.start, Level: f.heading})
		}
	}
	if f.isPagebreak {
		label := f.pageLabel
		if label == "" {
			label = normaliseSpaces(string(x.out[f.start:]))
		}
		if label != "" {
			x.pagebreaks = append(x.pagebreaks, Page{Label: label, Offset: f.start})
		}
	}
}

// hasTokenBytes reports whether a space-separated attribute carries a value.
func hasTokenBytes(attribute []byte, token string) bool {
	for len(attribute) > 0 {
		for len(attribute) > 0 && (attribute[0] == ' ' || attribute[0] == '\t' || attribute[0] == '\n' || attribute[0] == '\r') {
			attribute = attribute[1:]
		}
		end := 0
		for end < len(attribute) && attribute[end] != ' ' && attribute[end] != '\t' && attribute[end] != '\n' && attribute[end] != '\r' {
			end++
		}
		field := attribute[:end]
		attribute = attribute[end:]
		if strings.EqualFold(string(field), token) {
			return true
		}
	}
	return false
}

// isVoidTag reports whether an HTML element has no closing tag.
func isVoidTag(a atom.Atom) bool {
	switch a {
	case atom.Area, atom.Base, atom.Br, atom.Col, atom.Embed, atom.Hr,
		atom.Img, atom.Input, atom.Keygen, atom.Link, atom.Meta,
		atom.Param, atom.Source, atom.Track, atom.Wbr:
		return true
	}
	return false
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
		// gomutants:disable-next-line INTEGER_INCREMENT reason="the byte is read only when the second result is true"
		return 0, false
	}
	return x.out[len(x.out)-1], true
}

// getParts turns navigation entries into parts. An entry whose document is not in
// the book, or which carries no name, is not one.
func (x *extractor) getParts(entries []entry) []Part {
	var out []Part
	for _, e := range entries {
		title := normaliseSpaces(e.title)
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
		label := normaliseSpaces(e.title)
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
	// gomutants:disable-next-line INTEGER_DECREMENT reason="every caller tests the level for greater than zero"
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
