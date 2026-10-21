package epub

import (
	"archive/zip"
	"bytes"
	"path"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// An entry is one line of a navigation document: a name, and the href it points
// at.
type entry struct {
	title string
	href  string
}

// readNavigationParts reads the parts a book names, from whichever navigation
// document it carries. What counts is the entries that land in the text: a
// document whose entries do not is a document the book does not have, so the
// other one is read.
func readNavigationParts(files map[string]*zip.File, read packageDoc, text *extractor) []Part {
	if raw, ok := readArchiveEntry(files[read.ncx]); ok {
		if parts := text.getParts(readNCXNavMap(raw, path.Dir(read.ncx))); len(parts) >= minimumParts {
			return parts
		}
	}
	if raw, ok := readArchiveEntry(files[read.nav]); ok {
		return text.getParts(readNavDocument(raw, path.Dir(read.nav)))
	}
	return nil
}

// readPageEntries reads the pages of the printed book from the navigation control
// file's page list.
func readPageEntries(files map[string]*zip.File, read packageDoc) []entry {
	raw, _ := readArchiveEntry(files[read.ncx])
	return readNCXPageList(raw, path.Dir(read.ncx))
}

// An ncxPoint is a navPoint or a pageTarget. Both carry a label, a content
// element and, for a navPoint, children.
type ncxPoint struct {
	Value   string `xml:"value,attr"`
	Label   string `xml:"navLabel>text"`
	Content *struct {
		Src string `xml:"src,attr"`
	} `xml:"content"`
	Children []ncxPoint `xml:"navPoint"`
}

// readNCXNavMap reads the navigation control file's navMap, depth first, which is
// the order the entries are read in.
func readNCXNavMap(raw []byte, base string) []entry {
	var document struct {
		Points []ncxPoint `xml:"navMap>navPoint"`
	}
	if err := decodeXML(raw, &document); err != nil {
		return nil
	}
	var out []entry
	for _, point := range document.Points {
		out = appendPoint(out, point, base)
	}
	return out
}

// readNCXPageList reads the pages of the printed book. A page is named by its value
// when it has one; the label of a page target is often decoration around the
// number.
func readNCXPageList(raw []byte, base string) []entry {
	var document struct {
		Targets []ncxPoint `xml:"pageList>pageTarget"`
	}
	if err := decodeXML(raw, &document); err != nil {
		return nil
	}
	var out []entry
	for _, target := range document.Targets {
		if target.Content == nil {
			continue
		}
		name := target.Value
		if name == "" {
			name = target.Label
		}
		out = append(out, entry{title: name, href: resolveHref(base, target.Content.Src)})
	}
	return out
}

func appendPoint(out []entry, point ncxPoint, base string) []entry {
	// A navPoint may have no content element, and then it points nowhere.
	if point.Content != nil {
		out = append(out, entry{title: point.Label, href: resolveHref(base, point.Content.Src)})
	}
	for _, child := range point.Children {
		out = appendPoint(out, child, base)
	}
	return out
}

// readNavDocument reads an EPUB 3 navigation document: the links of its table of
// contents, in document order. A document with no table of contents is read for
// its links.
func readNavDocument(raw []byte, base string) []entry {
	root, err := html.Parse(bytes.NewReader(raw))
	// gomutants:disable-next-line BRANCH_IF reason="html.Parse on bytes.NewReader never errors"
	if err != nil {
		return nil
	}
	from := root
	if toc := findTableOfContents(root); toc != nil {
		from = toc
	}
	return appendLinks(nil, from, base)
}

// findTableOfContents returns the nav element marked as the table of contents,
// and nil when there is none.
func findTableOfContents(n *html.Node) *html.Node {
	if n.DataAtom == atom.Nav {
		if hasToken(getAttribute(n, "epub:type"), "toc") || hasToken(getAttribute(n, "role"), "doc-toc") {
			return n
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findTableOfContents(c); found != nil {
			return found
		}
	}
	return nil
}

func appendLinks(out []entry, n *html.Node, base string) []entry {
	if n.DataAtom == atom.A {
		if href := getAttribute(n, "href"); href != "" {
			out = append(out, entry{title: collectText(n), href: resolveHref(base, href)})
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = appendLinks(out, c, base)
	}
	return out
}

// collectText returns the text a node holds, whatever markup it is wrapped in.
func collectText(n *html.Node) string {
	var b bytes.Buffer
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// resolveHref resolves an href against the directory of the document holding it,
// keeping the fragment.
func resolveHref(base, href string) string {
	target, fragment := splitHref(href)
	at := resolve(base, target)
	if at == "" {
		return ""
	}
	if fragment == "" {
		return at
	}
	return at + "#" + fragment
}
