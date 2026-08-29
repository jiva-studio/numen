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

// navigationParts reads the parts a book names, from whichever navigation
// document it carries. What counts is the entries that land in the text: a
// document whose entries do not is a document the book does not have, so the
// other one is read.
func navigationParts(files map[string]*zip.File, read packageDoc, text *extractor) []Part {
	if raw, ok := contents(files[read.ncx]); ok {
		if parts := text.named(ncxNavMap(raw, path.Dir(read.ncx))); len(parts) >= minimumParts {
			return parts
		}
	}
	if raw, ok := contents(files[read.nav]); ok {
		return text.named(navDocument(raw, path.Dir(read.nav)))
	}
	return nil
}

// pageEntries reads the pages of the printed book from the navigation control
// file's page list.
func pageEntries(files map[string]*zip.File, read packageDoc) []entry {
	raw, ok := contents(files[read.ncx])
	if !ok {
		return nil
	}
	return ncxPageList(raw, path.Dir(read.ncx))
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

// ncxNavMap reads the navigation control file's navMap, depth first, which is
// the order the entries are read in.
func ncxNavMap(raw []byte, base string) []entry {
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

// ncxPageList reads the pages of the printed book. A page is named by its value
// when it has one; the label of a page target is often decoration around the
// number.
func ncxPageList(raw []byte, base string) []entry {
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
		out = append(out, entry{title: name, href: hrefIn(base, target.Content.Src)})
	}
	return out
}

func appendPoint(out []entry, point ncxPoint, base string) []entry {
	// A navPoint may have no content element, and then it points nowhere.
	if point.Content != nil {
		out = append(out, entry{title: point.Label, href: hrefIn(base, point.Content.Src)})
	}
	for _, child := range point.Children {
		out = appendPoint(out, child, base)
	}
	return out
}

// navDocument reads an EPUB 3 navigation document: the links of its table of
// contents, in document order. A document with no table of contents is read for
// its links.
func navDocument(raw []byte, base string) []entry {
	root, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	from := root
	if toc := tableOfContents(root); toc != nil {
		from = toc
	}
	return appendLinks(nil, from, base)
}

// tableOfContents is the nav element that says it is the table of contents.
func tableOfContents(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == atom.Nav {
		if hasToken(attribute(n, "epub:type"), "toc") || hasToken(attribute(n, "role"), "doc-toc") {
			return n
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := tableOfContents(c); found != nil {
			return found
		}
	}
	return nil
}

func appendLinks(out []entry, n *html.Node, base string) []entry {
	if n.Type == html.ElementNode && n.DataAtom == atom.A {
		if href := attribute(n, "href"); href != "" {
			out = append(out, entry{title: textOf(n), href: hrefIn(base, href)})
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = appendLinks(out, c, base)
	}
	return out
}

// textOf is the text a node holds, whatever markup it is wrapped in.
func textOf(n *html.Node) string {
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

// hrefIn resolves an href against the directory of the navigation document
// holding it, keeping the fragment.
func hrefIn(base, href string) string {
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
