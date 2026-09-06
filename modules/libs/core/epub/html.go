package epub

import (
	"html"
	"strconv"
	"strings"
)

// OffsetAttribute is what every run of text carries: where the run begins in the
// book's Text. A window reads an offset off the markup and counts none itself.
const OffsetAttribute = "data-offset"

// alone are the elements that hold nothing, and are written without a closing
// tag.
var alone = map[string]bool{"br": true, "col": true, "hr": true, "img": true}

// HTML is the document as the markup a window draws it from, every run of text
// wrapped in a span carrying its offset.
//
// It goes into a page unescaped, so the text and every attribute value is
// escaped here. The names and the attributes are the few a document is read
// down to, and escaping is the last thing between a book and the window's DOM.
func (m *Markup) HTML() string {
	var out strings.Builder
	out.Grow(2 * m.Length)
	write(&out, m.Nodes)
	return out.String()
}

func write(out *strings.Builder, nodes []Node) {
	for _, node := range nodes {
		if node.Name == "" {
			out.WriteString("<span " + OffsetAttribute + `="`)
			out.WriteString(strconv.Itoa(node.Offset))
			out.WriteString(`">`)
			out.WriteString(html.EscapeString(node.Text))
			out.WriteString("</span>")
			continue
		}
		out.WriteString("<" + node.Name)
		for _, a := range node.Attributes {
			out.WriteString(" " + a.Name + `="`)
			out.WriteString(html.EscapeString(a.Value))
			out.WriteString(`"`)
		}
		out.WriteString(">")
		if alone[node.Name] {
			continue
		}
		write(out, node.Children)
		out.WriteString("</" + node.Name + ">")
	}
}
