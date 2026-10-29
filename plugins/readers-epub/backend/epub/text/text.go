package text

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/href"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/words"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// An Extractor builds the one text of a book and remembers the offsets of the
// places the markup marks.
type Extractor struct {
	out []byte

	// path is the document being read, and starts holds the offset at which each
	// document begins, so an href with no fragment still resolves to an offset.
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

	// isPlacing records where each block element begins and the text node each text was written
	// from into places.
	isPlacing bool
	places    Places
}

func NewExtractor() *Extractor {
	return &Extractor{
		starts:  map[string]int{},
		anchors: map[string]int{},
	}
}

// ReadDocument reads one spine document into the text.
//
// The markup is parsed as HTML, so a document with three opening body tags is
// still readable.
func (x *Extractor) ReadDocument(docPath string, raw []byte) Document {
	start := len(x.out)
	x.path = docPath
	x.starts[docPath] = start

	if x.markup != nil {
		if root, err := html.Parse(bytes.NewReader(raw)); err == nil {
			x.writeChildren(root)
		}
	} else {
		x.tokenize(raw)
	}
	x.breakLine()
	return Document{Path: docPath, Offset: start, Length: len(x.out) - start}
}

func (x *Extractor) writeNode(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		x.writeTextNode(n)
		return
	case html.ElementNode:
	default:
		// gomutants:disable-next-line BRANCH_CASE reason="non-element nodes from html.Parse have no children and are not drawn"
		return
	}

	switch n.DataAtom {
	case atom.Head, atom.Script, atom.Style, atom.Title:
		// Not text of the book.
		return
	case atom.Br:
		x.breakLine()
		// begins.
		if x.markup.openElement(n, len(x.out)) {
			x.markup.close()
		}
		return
	}

	block := blocks[n.DataAtom]
	if block {
		x.breakLine()
		if x.isPlacing {
			x.places.Blocks = append(x.places.Blocks, len(x.out))
		}
	}

	if n.DataAtom == atom.Pre {
		x.preformattedDepth++
	}
	element := x.markup.openElement(n, len(x.out))
	if n.DataAtom == atom.Td || n.DataAtom == atom.Th {
		// A cell is set off from the one before it by a space, and the space
		x.write(" ")
	}
	x.writeChildren(n)
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

func (x *Extractor) writeChildren(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		x.writeNode(c)
	}
}

// write appends text. A run of space is one space, and text inside a pre element
// is kept as it was written.
func (x *Extractor) write(text string) {
	start := len(x.out)
	if x.preformattedDepth > 0 {
		x.out = append(x.out, text...)
	} else {
		x.writeNormalized(text)
	}
	if len(x.out) > start {
		x.markup.run(start)
	}
}

func (x *Extractor) writeNormalized(text string) {
	for _, r := range text {
		if unicode.IsSpace(r) {
			if last, ok := x.findLastByte(); ok && last != ' ' && last != '\n' {
				x.out = append(x.out, ' ')
			}
			continue
		}
		x.out = utf8.AppendRune(x.out, r)
	}
}

func (x *Extractor) writeBytes(b []byte) {
	if x.preformattedDepth > 0 {
		x.out = append(x.out, b...)
		return
	}
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if unicode.IsSpace(r) {
			if last, ok := x.findLastByte(); ok && last != ' ' && last != '\n' {
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

type tokenizer struct {
	x                  *Extractor
	z                  *html.Tokenizer
	stack              []frame
	skipAtom           atom.Atom
	dropLeadingNewline bool
}

type tagAttrs struct {
	id          string
	isPagebreak bool
	pageLabel   string
}

func (t *tokenizer) parseTagAttrs(hasAttr bool) tagAttrs {
	var attrs tagAttrs
	var titleAttr, ariaLabelAttr string
	for hasAttr {
		var key, val []byte
		key, val, hasAttr = t.z.TagAttr()
		switch {
		case bytes.Equal(key, []byte("id")):
			attrs.id = string(val)
		case bytes.Equal(key, []byte("epub:type")):
			if words.HasTokenBytes(val, "pagebreak") {
				attrs.isPagebreak = true
			}
		case bytes.Equal(key, []byte("role")):
			if words.HasTokenBytes(val, "doc-pagebreak") {
				attrs.isPagebreak = true
			}
		case bytes.Equal(key, []byte("title")):
			titleAttr = string(val)
		case bytes.Equal(key, []byte("aria-label")):
			ariaLabelAttr = string(val)
		}
	}
	if attrs.isPagebreak {
		if s := words.NormaliseSpaces(titleAttr); s != "" {
			attrs.pageLabel = s
		} else if al := words.NormaliseSpaces(ariaLabelAttr); al != "" {
			attrs.pageLabel = al
		}
	}
	return attrs
}

func (t *tokenizer) handleFinish() {
	for i := len(t.stack) - 1; i >= 0; i-- {
		t.x.closeFrame(t.stack[i])
	}
	t.x.preformattedDepth = 0
}

func (t *tokenizer) handleText() {
	if t.skipAtom == 0 {
		text := t.z.Text()
		if t.dropLeadingNewline {
			text = bytes.TrimPrefix(bytes.TrimPrefix(text, []byte{'\r'}), []byte{'\n'})
		}
		t.x.writeBytes(text)
	}
	t.dropLeadingNewline = false
}

func (t *tokenizer) shouldSkipStartTag(tt html.TokenType, a atom.Atom) bool {
	if t.skipAtom != 0 {
		if blocks[a] {
			t.skipAtom = 0
		} else {
			return true
		}
	}
	if a == atom.Head || a == atom.Script || a == atom.Style || a == atom.Title {
		if tt == html.StartTagToken {
			t.skipAtom = a
		}
		return true
	}
	return false
}

func (t *tokenizer) adjustElementState(a atom.Atom) {
	if a == atom.Pre {
		t.x.preformattedDepth++
		t.dropLeadingNewline = true
	}
	if a == atom.Listing || a == atom.Textarea {
		t.dropLeadingNewline = true
	}
	if a == atom.Td || a == atom.Th {
		t.x.writeBytes([]byte(" "))
	}
}

func (t *tokenizer) handleStartTag(tt html.TokenType) {
	t.dropLeadingNewline = false
	name, hasAttr := t.z.TagName()
	a := atom.Lookup(name)

	if t.shouldSkipStartTag(tt, a) {
		return
	}
	if a == atom.Br {
		t.x.breakLine()
		return
	}

	block := blocks[a]
	if block {
		t.x.breakLine()
	}

	attrs := t.parseTagAttrs(hasAttr)
	if attrs.id != "" {
		t.x.anchors[t.x.path+"#"+attrs.id] = len(t.x.out)
	}

	start := len(t.x.out)
	t.adjustElementState(a)

	level := parseHeadingLevel(a)
	if isVoidTag(a) {
		t.x.closeFrame(frame{start: start, isPagebreak: attrs.isPagebreak, pageLabel: attrs.pageLabel})
		// gomutants:disable-next-line BRANCH_IF reason="the preceding line break leaves the buffer ending in a newline"
		if block {
			t.x.breakLine()
		}
		return
	}

	t.stack = append(t.stack, frame{
		atom:        a,
		start:       start,
		block:       block,
		heading:     level,
		isPagebreak: attrs.isPagebreak,
		pageLabel:   attrs.pageLabel,
	})
}

func (t *tokenizer) findStackFrame(a atom.Atom) int {
	for i := len(t.stack) - 1; i >= 0; i-- {
		if t.stack[i].atom == a {
			return i
		}
	}
	return -1
}

func (t *tokenizer) popFramesUntil(target int) {
	for i := len(t.stack) - 1; i >= target; i-- {
		t.x.closeFrame(t.stack[i])
		if t.stack[i].block {
			t.x.breakLine()
		}
	}
	t.stack = t.stack[:target]
}

func (t *tokenizer) handleEndTag() {
	name, _ := t.z.TagName()
	a := atom.Lookup(name)

	if t.skipAtom != 0 {
		if a == t.skipAtom {
			t.skipAtom = 0
		}
		return
	}

	if a == atom.Br || a == atom.Body || a == atom.Html {
		if a == atom.Br {
			t.x.breakLine()
		}
		return
	}

	found := t.findStackFrame(a)
	if found >= 0 {
		t.popFramesUntil(found)
	} else if blocks[a] {
		t.x.breakLine()
	}
}

// tokenize reads markup into text using a streaming HTML tokenizer.
func (x *Extractor) tokenize(raw []byte) {
	t := tokenizer{
		x: x,
		z: html.NewTokenizer(bytes.NewReader(raw)),
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="the capacity does not change the slice's contents"
		stack:              make([]frame, 0, 32),
		skipAtom:           0,
		dropLeadingNewline: false,
	}

	for {
		tt := t.z.Next()
		switch tt {
		case html.ErrorToken:
			t.handleFinish()
			return
		case html.CommentToken, html.DoctypeToken:
		case html.TextToken:
			t.handleText()
		case html.StartTagToken, html.SelfClosingTagToken:
			t.handleStartTag(tt)
		case html.EndTagToken:
			t.handleEndTag()
		}
	}
}

// closeFrame records the heading and the page break an element that has ended
// carried, and leaves a pre element.
func (x *Extractor) closeFrame(f frame) {
	if f.atom == atom.Pre {
		x.preformattedDepth--
	}
	if f.heading > 0 {
		if title := words.NormaliseSpaces(string(x.out[f.start:])); title != "" {
			x.headings = append(x.headings, Part{Title: title, Offset: f.start, Level: f.heading})
		}
	}
	if f.isPagebreak {
		label := f.pageLabel
		if label == "" {
			label = words.NormaliseSpaces(string(x.out[f.start:]))
		}
		if label != "" {
			x.pagebreaks = append(x.pagebreaks, Page{Label: label, Offset: f.start})
		}
	}
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

func (x *Extractor) breakLine() {
	if last, ok := x.findLastByte(); ok && last == ' ' {
		x.out = x.out[:len(x.out)-1]
	}
	if last, ok := x.findLastByte(); ok && last != '\n' {
		start := len(x.out)
		x.out = append(x.out, '\n')
		x.markup.run(start)
	}
}

func (x *Extractor) findLastByte() (byte, bool) {
	if len(x.out) == 0 {
		// gomutants:disable-next-line INTEGER_INCREMENT reason="the byte is read only when the second result is true"
		return 0, false
	}
	return x.out[len(x.out)-1], true
}

// GetParts turns navigation entries into parts. An entry whose document is not in
// the book, or which carries no name, is not one.
func (x *Extractor) GetParts(entries []Entry) []Part {
	var out []Part
	for _, e := range entries {
		title := words.NormaliseSpaces(e.Title)
		if title == "" {
			continue
		}
		offset, ok := x.findOffset(e.Href)
		if !ok {
			continue
		}
		out = append(out, Part{Title: title, Offset: offset})
	}
	return out
}

// GetPages turns the entries of a page list into pages. A book with no page list
// keeps the page breaks its markup marks.
func (x *Extractor) GetPages(entries []Entry) []Page {
	var out []Page
	for _, e := range entries {
		label := words.NormaliseSpaces(e.Title)
		if label == "" {
			continue
		}
		offset, ok := x.findOffset(e.Href)
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

// findOffset returns the text offset an href names: its fragment's anchor, or
// the start of its document. The result is false when neither is known.
func (x *Extractor) findOffset(ref string) (int, bool) {
	target, fragment := href.Split(ref)
	if fragment != "" {
		if offset, ok := x.anchors[target+"#"+fragment]; ok {
			return offset, true
		}
	}
	offset, ok := x.starts[target]
	return offset, ok
}

// GetAttribute reads an attribute of the element by name. The markup is read as
// HTML, where a prefix is part of the name: `epub:type` is an attribute called
// "epub:type".
func GetAttribute(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
		}
	}
	return ""
}

// parseHeadingLevel returns the level of a heading element, and 0 for anything else.
func parseHeadingLevel(a atom.Atom) int {
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

// blocks are the elements whose text relies on a line of its own.
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

// GetText is the text of every document read so far.
func (x *Extractor) GetText() []byte {
	return x.out
}

// GetHeadings is the parts the headings of the markup name.
func (x *Extractor) GetHeadings() []Part {
	return x.headings
}
