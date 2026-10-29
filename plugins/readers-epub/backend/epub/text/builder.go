package text

import (
	"path"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/href"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/markup"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// A builder gathers the elements of one document as its text is read.
//
// An element that is not drawn keeps its text and loses itself, so the words
// here are the words of the text and neither can drift from the other. The four
// the text is not taken from — head, script, style and title — are dropped whole
// by the walk that reads both.
type builder struct {
	// written against.
	base string
	// stack is the elements standing open, the first being the document itself.
	stack []markup.Node
	// waiting are the runs written in a table before its first cell, which the
	// cell takes when it opens.
	waiting []markup.Node
}

func newBuilder(base string) *builder {
	return &builder{base: base, stack: []markup.Node{{}}}
}

// openElement starts an element and reports whether one was started. An element that
// is not drawn is passed through to its children.
func (b *builder) openElement(n *html.Node, offset int) bool {
	if !drawn[n.DataAtom] {
		return false
	}
	opening := markup.Node{Name: n.Data, Offset: offset, Attributes: b.convertAttributes(n)}
	if cells[opening.Name] && len(b.waiting) > 0 {
		opening.Children, opening.Offset = b.waiting, b.waiting[0].Offset
		b.waiting = nil
	}
	b.stack = append(b.stack, opening)
	return true
}

func (b *builder) close() {
	isDone := b.stack[len(b.stack)-1]
	b.stack = b.stack[:len(b.stack)-1]
	// A table that opened no cell keeps the runs written inside it, in the
	// order they stand.
	if len(b.waiting) > 0 && tabular[isDone.Name] && !tabular[b.stack[len(b.stack)-1].Name] {
		isDone.Children, isDone.Offset = append(b.waiting, isDone.Children...), b.waiting[0].Offset
		b.waiting = nil
	}
	b.appendChild(isDone)
}

// run marks a run of text written at an offset, whether written from the markup or
// to end a line. Its content is filled in when the document has been read, so a space
// removed by the following line is removed here too.
//
// one standing between the cells out in front of the table.
func (b *builder) run(offset int) {
	if b == nil {
		return
	}
	top := &b.stack[len(b.stack)-1]
	switch cell := findLastCell(top); {
	case cell != nil:
		cell.Children = append(cell.Children, markup.Node{Offset: offset})
	case tabular[top.Name]:
		b.waiting = append(b.waiting, markup.Node{Offset: offset})
	default:
		b.appendChild(markup.Node{Offset: offset})
	}
}

// findLastCell returns the last cell opened under an element, and nil where the
// element is no part of a table or holds no cell yet.
func findLastCell(in *markup.Node) *markup.Node {
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

func (b *builder) appendChild(n markup.Node) {
	top := &b.stack[len(b.stack)-1]
	top.Children = append(top.Children, n)
}

// document.
//
// A run extends to where the next one begins, so every byte of the document is
// covered exactly once.
func (b *builder) finish(text string, end int) []markup.Node {
	nodes := b.stack[0].Children
	found := collectNodes(nodes, nil)

	// The line a block ends removes the space before it, and an element on that space
	// moves to where the line ends.
	pos := end
	for i := len(found) - 1; i >= 0; i-- {
		found[i].Offset = min(found[i].Offset, pos)
		pos = found[i].Offset
	}

	var last *markup.Node
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

// collectNodes appends the nodes of a finished document to the list, in reading order.
func collectNodes(nodes []markup.Node, into []*markup.Node) []*markup.Node {
	for i := range nodes {
		into = append(into, &nodes[i])
		into = collectNodes(nodes[i].Children, into)
	}
	return into
}

// convertAttributes returns the attributes of an element that survive, with href
// and src resolved. A book's own styling is not among them.
func (b *builder) convertAttributes(n *html.Node) []markup.Attribute {
	var out []markup.Attribute
	for _, a := range n.Attr {
		name := strings.ToLower(a.Key)
		if !anyElement[name] && !slices.Contains(ownAttributes[n.DataAtom], name) {
			continue
		}
		value := a.Val
		switch name {
		case "href":
			if value = resolveLinkAddress(b.base, value); value == "" {
				continue
			}
		case "src":
			if value = resolveImageSource(b.base, value); value == "" {
				continue
			}
		}
		out = append(out, markup.Attribute{Name: name, Value: value})
	}
	return out
}

// resolveLinkAddress resolves a link to the archive entry it names and the place
// in it, or to an address in one of the reachable schemes. Anything else
// resolves to empty and the attribute goes.
//
// An href naming no document names a place in the document being read, and
// crosses as the fragment it is.
func resolveLinkAddress(base, url string) string {
	if named := parseScheme(removeSpacesAndControls(url)); named != "" {
		if reachable[named] {
			return removeSpacesAndControls(url)
		}
		return ""
	}
	if target, fragment := href.Split(url); target == "" {
		if fragment == "" {
			return ""
		}
		return "#" + fragment
	}
	return href.ResolveWithFragment(base, url)
}

// resolveImageSource resolves an img source to a file the book carries or to an
// inline base64 image. An address off the machine resolves to empty.
func resolveImageSource(base, url string) string {
	if parseScheme(removeSpacesAndControls(url)) == "" {
		return href.ResolveWithFragment(base, url)
	}
	if isInlineImage(removeSpacesAndControls(url)) {
		return removeSpacesAndControls(url)
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

// isInlineImage reports whether an address is a base64 image written into the markup.
func isInlineImage(url string) bool {
	head, _, content := strings.Cut(strings.ToLower(url), ",")
	return content && strings.HasPrefix(head, "data:image/") && strings.HasSuffix(head, ";base64")
}

// removeSpacesAndControls returns an address with its spaces and control
// characters removed.
func removeSpacesAndControls(url string) string {
	return strings.Map(func(r rune) rune {
		if r <= ' ' {
			// gomutants:disable-next-line INTEGER_INCREMENT reason="strings.Map drops a rune for any negative result, -2 included"
			return -1
		}
		return r
	}, url)
}

// parseScheme returns the scheme a URL names, and empty when it names none.
func parseScheme(url string) string {
	for i := 0; i < len(url); i++ {
		c := url[i]
		switch {
		case c == ':':
			return url[:i]
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

// ReadNodes reads one document as the elements it is drawn from. The offsets of
// the nodes count from the start of the document.
func ReadNodes(docPath string, raw []byte) []markup.Node {
	x := NewExtractor()
	x.markup = newBuilder(path.Dir(docPath))
	read := x.ReadDocument(docPath, raw)
	return x.markup.finish(string(x.out), read.Length)
}
