package text

import (
	"bytes"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// An Extractor builds the one text of a book and remembers the offsets of the
// places the markup marks.
type Extractor struct {
	out []byte

	path   string
	starts map[string]int

	anchors map[string]int

	headings   []Part
	pagebreaks []Page

	markup *builder

	preformattedDepth int

	isPlacing bool
	places    Places
}

// NewExtractor returns an Extractor with no document read.
func NewExtractor() *Extractor {
	return &Extractor{
		out:               nil,
		path:              "",
		starts:            map[string]int{},
		anchors:           map[string]int{},
		headings:          nil,
		pagebreaks:        nil,
		markup:            nil,
		preformattedDepth: 0,
		isPlacing:         false,
		places:            Places{Blocks: nil, Runs: nil},
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
	return Document{Path: docPath, Offset: start, Length: len(x.out) - start, IsLinear: false, Layout: "", Spine: 0}
}

func (x *Extractor) writeNode(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		x.writeTextNode(n)
		return
	case html.ElementNode:
	case html.ErrorNode, html.DocumentNode, html.CommentNode, html.DoctypeNode, html.RawNode:
		// gomutants:disable-next-line BRANCH_CASE reason="non-element nodes from html.Parse have no children and are not drawn"
		return
	}

	if isSkipped(n.DataAtom) {
		// Not text of the book.
		return
	}
	if n.DataAtom == atom.Br {
		x.writeLineBreak(n)
		return
	}

	block := isBlock(n.DataAtom)
	if block {
		x.startBlock()
	}

	if n.DataAtom == atom.Pre {
		x.preformattedDepth++
	}
	element := x.markup.openElement(n, len(x.out))
	x.writeCellGap(n)
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

func (x *Extractor) writeCellGap(n *html.Node) {
	if n.DataAtom == atom.Td || n.DataAtom == atom.Th {
		x.write(" ")
	}
}

func (x *Extractor) writeLineBreak(n *html.Node) {
	x.breakLine()
	// begins.
	if x.markup.openElement(n, len(x.out)) {
		x.markup.close()
	}
}

func (x *Extractor) startBlock() {
	x.breakLine()
	if x.isPlacing {
		x.places.Blocks = append(x.places.Blocks, len(x.out))
	}
}

func (x *Extractor) writeChildren(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		x.writeNode(c)
	}
}

func (x *Extractor) write(text string) {
	start := len(x.out)
	if x.preformattedDepth > 0 {
		x.out = append(x.out, text...)
	} else {
		x.writeNormalizedText(text)
	}
	if len(x.out) > start {
		x.markup.run(start)
	}
}

func (x *Extractor) writeNormalizedText(text string) {
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

// GetText is the text of every document read so far.
func (x *Extractor) GetText() []byte {
	return x.out
}

// GetHeadings is the parts the headings of the markup name.
func (x *Extractor) GetHeadings() []Part {
	return x.headings
}
