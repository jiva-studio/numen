package epub

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// A Markup is one spine document as the elements a book is drawn from, with
// every run of text at the offset it stands at in the book's Text. Reading a
// document and reading its text are one walk, so an offset here is the offset a
// chunk of that text carries.
type Markup struct {
	// Path is the document's name inside the archive.
	Path string
	// Offset is where the document's text begins in the book's Text, and Length
	// how many bytes of it the document is.
	Offset int
	Length int
	// Nodes are what the document holds, in reading order.
	Nodes []Node
}

// A Node is one element of a document, or one run of its text.
type Node struct {
	// Name is the element's name, and empty for a run of text.
	Name string
	// Text is the run's text, and empty for an element.
	Text string
	// Offset is where the node begins in the book's Text.
	Offset     int
	Attributes []Attribute
	Children   []Node
}

// An Attribute is one attribute of an element, of the few an element may carry.
type Attribute struct {
	Name  string
	Value string
}

// Markup reads one spine document again, as elements rather than as text.
//
// A document the book was not read from is not one to ask for: what a book holds
// beside its spine is not part of it.
func (b *Book) Markup(docPath string) (*Markup, error) {
	doc, ok := b.held[docPath]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoDocument, docPath)
	}
	raw, ok := within(b.files[docPath], mostPerDocument)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoDocument, docPath)
	}

	x := newExtractor()
	x.markup = newBuilder(path.Dir(docPath))
	read := x.document(docPath, raw)
	nodes := x.markup.finish(string(x.out), read.Length)

	// A spine document the manifest calls an SVG is a cover drawn around a
	// picture, and what a reader draws is that picture. One that draws none is
	// answered with the text it carries and nothing else.
	if doc.mediaType == mediaSVG {
		if at, ok := b.findSVGImage(path.Dir(docPath), raw); ok {
			nodes = append([]Node{{Name: "img", Attributes: []Attribute{{Name: "src", Value: at}}}}, nodes...)
		}
	}
	shift(nodes, doc.Offset)
	return &Markup{Path: docPath, Offset: doc.Offset, Length: doc.Length, Nodes: nodes}, nil
}

// findSVGImage is the picture an SVG is drawn around: the first image element of it
// naming an entry the archive holds, or one written into the file itself.
func (b *Book) findSVGImage(base string, raw []byte) (string, bool) {
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	decoder.Strict = false
	decoder.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil }
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", false
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "image" {
			continue
		}
		for _, a := range start.Attr {
			// An SVG names what it draws with href, and with xlink:href in the
			// version most books are written in.
			if a.Name.Local != "href" {
				continue
			}
			switch at := picture(base, a.Value); {
			case at == "":
			case inlineImage(at), b.files[at] != nil:
				return at, true
			}
		}
	}
}

// A builder gathers the elements of one document as its text is read.
//
// An element that is not drawn keeps its text and loses itself, so the words
// here are the words of the text and neither can drift from the other. The four
// the text is not taken from — head, script, style and title — are dropped whole
// by the walk that reads both.
type builder struct {
	// base is the directory the document sits in, which its own hrefs are
	// written against.
	base string
	// stack is the elements standing open, the first being the document itself.
	stack []Node
	// waiting are the runs written in a table before its first cell, which the
	// cell takes when it opens.
	waiting []Node
}

func newBuilder(base string) *builder {
	return &builder{base: base, stack: []Node{{}}}
}

// openElement starts an element and reports whether one was started. An element that
// is not drawn is passed through to its children.
func (b *builder) openElement(n *html.Node, offset int) bool {
	if b == nil || !drawn[n.DataAtom] {
		return false
	}
	opening := Node{Name: n.Data, Offset: offset, Attributes: b.attributes(n)}
	if cells[opening.Name] && len(b.waiting) > 0 {
		opening.Children, opening.Offset = b.waiting, b.waiting[0].Offset
		b.waiting = nil
	}
	b.stack = append(b.stack, opening)
	return true
}

func (b *builder) close() {
	if b == nil {
		return
	}
	done := b.stack[len(b.stack)-1]
	b.stack = b.stack[:len(b.stack)-1]
	// A table that opened no cell keeps the runs written inside it, in the
	// order they stand.
	if len(b.waiting) > 0 && tabular[done.Name] && !tabular[b.stack[len(b.stack)-1].Name] {
		done.Children, done.Offset = append(b.waiting, done.Children...), b.waiting[0].Offset
		b.waiting = nil
	}
	b.hold(done)
}

// run marks a run of text written at an offset, whether it was written from the
// markup or to end a line. What it says is filled in when the document has been
// read, so a space taken back by the line that follows is taken back here too.
//
// A run written in a table stands inside a cell of it: a browser foster-parents
// one standing between the cells out in front of the table.
func (b *builder) run(offset int) {
	if b == nil {
		return
	}
	top := &b.stack[len(b.stack)-1]
	switch cell := lastCell(top); {
	case cell != nil:
		cell.Children = append(cell.Children, Node{Offset: offset})
	case tabular[top.Name]:
		b.waiting = append(b.waiting, Node{Offset: offset})
	default:
		b.hold(Node{Offset: offset})
	}
}

// lastCell is the last cell opened under an element, and nil where the element
// is no part of a table or holds no cell yet.
func lastCell(in *Node) *Node {
	for tabular[in.Name] && len(in.Children) > 0 {
		last := &in.Children[len(in.Children)-1]
		if cells[last.Name] {
			return last
		}
		in = last
	}
	return nil
}

// tabular are the elements that hold rows and cells and no text of their own,
// and cells are the ones that hold text.
var (
	tabular = map[string]bool{
		"table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true,
	}
	cells = map[string]bool{"td": true, "th": true}
)

func (b *builder) hold(n Node) {
	top := &b.stack[len(b.stack)-1]
	top.Children = append(top.Children, n)
}

// finish fills every run with the text it stands over and answers with the
// document.
//
// A run reaches to where the next one begins, so every byte of the document is
// said once and in one place.
func (b *builder) finish(text string, end int) []Node {
	for len(b.stack) > 1 {
		b.close()
	}
	nodes := b.stack[0].Children
	found := nodesIn(nodes, nil)

	// The line a block ends takes back the space before it, and an element
	// standing on that space stands where the line ends instead.
	at := end
	for i := len(found) - 1; i >= 0; i-- {
		found[i].Offset = min(found[i].Offset, at)
		at = found[i].Offset
	}

	var last *Node
	for _, node := range found {
		if node.Name != "" {
			continue
		}
		if last != nil {
			last.Text = text[last.Offset:node.Offset]
		}
		last = node
	}
	if last != nil {
		last.Text = text[last.Offset:end]
	}
	return nodes
}

// nodesIn are the nodes of a finished document, in reading order.
func nodesIn(nodes []Node, into []*Node) []*Node {
	for i := range nodes {
		into = append(into, &nodes[i])
		into = nodesIn(nodes[i].Children, into)
	}
	return into
}

// shift moves a document to where it stands in the book.
func shift(nodes []Node, by int) {
	for i := range nodes {
		nodes[i].Offset += by
		shift(nodes[i].Children, by)
	}
}

// attributes are the attributes of an element that survive. A book's own styling
// is not among them: how a book looks belongs to the window it is read in.
func (b *builder) attributes(n *html.Node) []Attribute {
	var out []Attribute
	for _, a := range n.Attr {
		name := strings.ToLower(a.Key)
		if !anyElement[name] && !slices.Contains(ownAttributes[n.DataAtom], name) {
			continue
		}
		value := a.Val
		switch name {
		case "href":
			if value = address(b.base, value); value == "" {
				continue
			}
		case "src":
			if value = picture(b.base, value); value == "" {
				continue
			}
		}
		out = append(out, Attribute{Name: name, Value: value})
	}
	return out
}

// address is where a link points: inside the book, the archive entry it names
// and the place in it; outward, an address in one of the few schemes a person
// may be sent to. Anything else leads nowhere and the attribute goes.
//
// An href naming no document names a place in the document being read, and
// crosses as the fragment it is.
func address(base, said string) string {
	if named := scheme(bare(said)); named != "" {
		if reachable[named] {
			return bare(said)
		}
		return ""
	}
	if target, fragment := splitHref(said); target == "" {
		if fragment == "" {
			return ""
		}
		return "#" + fragment
	}
	return hrefIn(base, said)
}

// picture is what an img draws: a file the book carries, or the bytes written
// into the markup itself. An address off the machine is a request made the
// moment the book is opened, which tells whoever wrote the book that it was
// read, and from where.
func picture(base, said string) string {
	if scheme(bare(said)) == "" {
		return hrefIn(base, said)
	}
	if inlineImage(bare(said)) {
		return bare(said)
	}
	return ""
}

// reachable are the schemes a link may name.
var reachable = map[string]bool{
	"http":   true,
	"https":  true,
	"mailto": true,
	"tel":    true,
}

// inlineImage is whether an address is a picture written into the markup itself.
func inlineImage(said string) bool {
	head, _, written := strings.Cut(strings.ToLower(said), ",")
	return written && strings.HasPrefix(head, "data:image/") && strings.HasSuffix(head, ";base64")
}

// bare is an address with the spaces and control characters a scheme can be
// hidden behind taken out. A browser reads one out of them all the same.
func bare(said string) string {
	return strings.Map(func(r rune) rune {
		if r <= ' ' {
			return -1
		}
		return r
	}, said)
}

// scheme is the scheme a URL names, and empty for one that names none.
func scheme(said string) string {
	for i := 0; i < len(said); i++ {
		c := said[i]
		switch {
		case c == ':':
			if i == 0 {
				return ""
			}
			return said[:i]
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case i > 0 && (c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.'):
		default:
			return ""
		}
	}
	return ""
}

// anyElement are the attributes any element may carry. An id is how a link
// inside the book lands on the element it names.
var anyElement = map[string]bool{
	"id":    true,
	"dir":   true,
	"lang":  true,
	"title": true,
}

// ownAttributes are what an element may carry beyond those.
var ownAttributes = map[atom.Atom][]string{
	atom.A:        {"href"},
	atom.Img:      {"src", "alt", "width", "height"},
	atom.Td:       {"colspan", "rowspan"},
	atom.Th:       {"colspan", "rowspan", "scope"},
	atom.Ol:       {"start", "reversed"},
	atom.Time:     {"datetime"},
	atom.Col:      {"span"},
	atom.Colgroup: {"span"},
}

// drawn are the elements a book is drawn from.
var drawn = map[atom.Atom]bool{
	atom.A:          true,
	atom.Abbr:       true,
	atom.Address:    true,
	atom.Article:    true,
	atom.Aside:      true,
	atom.B:          true,
	atom.Bdi:        true,
	atom.Bdo:        true,
	atom.Blockquote: true,
	atom.Br:         true,
	atom.Caption:    true,
	atom.Cite:       true,
	atom.Code:       true,
	atom.Col:        true,
	atom.Colgroup:   true,
	atom.Dd:         true,
	atom.Del:        true,
	atom.Dfn:        true,
	atom.Div:        true,
	atom.Dl:         true,
	atom.Dt:         true,
	atom.Em:         true,
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
	atom.I:          true,
	atom.Img:        true,
	atom.Ins:        true,
	atom.Kbd:        true,
	atom.Li:         true,
	atom.Main:       true,
	atom.Mark:       true,
	atom.Nav:        true,
	atom.Ol:         true,
	atom.P:          true,
	atom.Pre:        true,
	atom.Q:          true,
	atom.Rp:         true,
	atom.Rt:         true,
	atom.Ruby:       true,
	atom.S:          true,
	atom.Samp:       true,
	atom.Section:    true,
	atom.Small:      true,
	atom.Span:       true,
	atom.Strong:     true,
	atom.Sub:        true,
	atom.Sup:        true,
	atom.Table:      true,
	atom.Tbody:      true,
	atom.Td:         true,
	atom.Tfoot:      true,
	atom.Th:         true,
	atom.Thead:      true,
	atom.Time:       true,
	atom.Tr:         true,
	atom.U:          true,
	atom.Ul:         true,
	atom.Var:        true,
}
