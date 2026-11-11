// Package text extracts the text, parts, pages, places and markup of EPUB documents.
package text

import (
	"path"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/domain/epub/href"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/domain/epub/markup"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// hrefAttribute is the name of the attribute that holds a link address.
const hrefAttribute = "href"

// A builder gathers the elements of one document as its text is read.
//
// An element that is not drawn keeps its text and loses itself, so the words
// here are the words of the text and neither can drift from the other. The four
// the text is not taken from — head, script, style and title — are dropped whole
// by the walk that reads both.
type builder struct {
	base    string
	stack   []markup.Node
	waiting []markup.Node
}

func newBuilder(base string) *builder {
	return &builder{base: base, stack: []markup.Node{{Name: "", Text: "", Offset: 0, Attributes: nil, Children: nil}}, waiting: nil}
}

func (b *builder) openElement(n *html.Node, offset int) bool {
	if !isDrawn(n.DataAtom) {
		return false
	}
	opening := markup.Node{Name: n.Data, Text: "", Offset: offset, Attributes: b.convertAttributes(n), Children: nil}
	if isCell(opening.Name) && len(b.waiting) > 0 {
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
	if len(b.waiting) > 0 && isTabular(isDone.Name) && !isTabular(b.stack[len(b.stack)-1].Name) {
		isDone.Children, isDone.Offset = append(b.waiting, isDone.Children...), b.waiting[0].Offset
		b.waiting = nil
	}
	b.appendChild(isDone)
}

func (b *builder) run(offset int) {
	if b == nil {
		return
	}
	top := &b.stack[len(b.stack)-1]
	switch cell := findLastCell(top); {
	case cell != nil:
		cell.Children = append(cell.Children, markup.Node{Name: "", Text: "", Offset: offset, Attributes: nil, Children: nil})
	case isTabular(top.Name):
		b.waiting = append(b.waiting, markup.Node{Name: "", Text: "", Offset: offset, Attributes: nil, Children: nil})
	default:
		b.appendChild(markup.Node{Name: "", Text: "", Offset: offset, Attributes: nil, Children: nil})
	}
}

func findLastCell(in *markup.Node) *markup.Node {
	for isTabular(in.Name) && len(in.Children) > 0 {
		last := &in.Children[len(in.Children)-1]
		if isCell(last.Name) {
			return last
		}
		in = last
	}
	return nil
}

func isTabular(name string) bool {
	switch name {
	case "table", "thead", "tbody", "tfoot", "tr":
		return true
	}
	return false
}

func isCell(name string) bool {
	return name == "td" || name == "th"
}

func (b *builder) appendChild(n markup.Node) {
	top := &b.stack[len(b.stack)-1]
	top.Children = append(top.Children, n)
}

func (b *builder) finish(text string, end int) []markup.Node {
	nodes := b.stack[0].Children
	found := collectNodes(nodes, nil)

	// The line a block ends removes the space before it, and an element on that space
	// moves to where the line ends.
	pos := end
	for _, f := range slices.Backward(found) {
		f.Offset = min(f.Offset, pos)
		pos = f.Offset
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

func collectNodes(nodes []markup.Node, into []*markup.Node) []*markup.Node {
	for i := range nodes {
		into = append(into, &nodes[i])
		into = collectNodes(nodes[i].Children, into)
	}
	return into
}

func (b *builder) convertAttributes(n *html.Node) []markup.Attribute {
	var out []markup.Attribute
	for _, a := range n.Attr {
		name := strings.ToLower(a.Key)
		if !isAnyElementAttribute(name) && !slices.Contains(getOwnAttributes(n.DataAtom), name) {
			continue
		}
		value := a.Val
		switch name {
		case hrefAttribute:
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

func resolveLinkAddress(base, url string) string {
	if named := parseScheme(removeSpacesAndControls(url)); named != "" {
		if isReachable(named) {
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

func resolveImageSource(base, url string) string {
	if parseScheme(removeSpacesAndControls(url)) == "" {
		return href.ResolveWithFragment(base, url)
	}
	if isInlineImage(removeSpacesAndControls(url)) {
		return removeSpacesAndControls(url)
	}
	return ""
}

func isReachable(key string) bool {
	switch key {
	case "http", "https", "mailto", "tel":
		return true
	}
	return false
}

func isInlineImage(url string) bool {
	head, _, content := strings.Cut(strings.ToLower(url), ",")
	return content && strings.HasPrefix(head, "data:image/") && strings.HasSuffix(head, ";base64")
}

func removeSpacesAndControls(url string) string {
	return strings.Map(func(r rune) rune {
		if r <= ' ' {
			// gomutants:disable-next-line INTEGER_INCREMENT reason="strings.Map drops a rune for any negative result, -2 included"
			return -1
		}
		return r
	}, url)
}

func parseScheme(url string) string {
	for i := range len(url) {
		c := url[i]
		if c == ':' {
			return url[:i]
		}
		if !isSchemeByte(c, i) {
			return ""
		}
	}
	return ""
}

func isSchemeByte(c byte, i int) bool {
	if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
		return true
	}
	return i > 0 && (c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.')
}

func isAnyElementAttribute(key string) bool {
	switch key {
	case "id", "dir", "lang", "title":
		return true
	}
	return false
}

func getOwnAttributes(element atom.Atom) []string {
	switch element {
	case atom.A:
		return []string{hrefAttribute}
	case atom.Img:
		return []string{"src", "alt", "width", "height"}
	case atom.Td:
		return []string{"colspan", "rowspan"}
	case atom.Th:
		return []string{"colspan", "rowspan", "scope"}
	case atom.Ol:
		return []string{"start", "reversed"}
	case atom.Time:
		return []string{"datetime"}
	case atom.Col, atom.Colgroup:
		return []string{"span"}
	}
	return nil
}

func isDrawn(key atom.Atom) bool {
	switch key {
	case atom.A, atom.Abbr, atom.Address, atom.Article, atom.Aside, atom.B, atom.Bdi, atom.Bdo,
		atom.Blockquote, atom.Br, atom.Caption, atom.Cite, atom.Code, atom.Col, atom.Colgroup, atom.Dd,
		atom.Del, atom.Dfn, atom.Div, atom.Dl, atom.Dt, atom.Em, atom.Figcaption, atom.Figure,
		atom.Footer, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Header, atom.Hr, atom.I,
		atom.Img, atom.Ins, atom.Kbd, atom.Li, atom.Main, atom.Mark, atom.Nav, atom.Ol, atom.P, atom.Pre,
		atom.Q, atom.Rp, atom.Rt, atom.Ruby, atom.S, atom.Samp, atom.Section, atom.Small, atom.Span,
		atom.Strong, atom.Sub, atom.Sup, atom.Table, atom.Tbody, atom.Td, atom.Tfoot, atom.Th, atom.Thead,
		atom.Time, atom.Tr, atom.U, atom.Ul, atom.Var:
		return true
	}
	return false
}

// ReadNodes reads one document as the elements it is drawn from. The offsets of
// the nodes count from the start of the document.
func ReadNodes(docPath string, raw []byte) []markup.Node {
	x := NewExtractor()
	x.markup = newBuilder(path.Dir(docPath))
	read := x.ReadDocument(docPath, raw)
	return x.markup.finish(string(x.out), read.Length)
}
