package navigation

import (
	"archive/zip"
	"bytes"
	"path"

	"github.com/jiva-studio/numen/modules/plugins/readers.epub/backend/epub/archive"
	"github.com/jiva-studio/numen/modules/plugins/readers.epub/backend/epub/href"
	"github.com/jiva-studio/numen/modules/plugins/readers.epub/backend/epub/opf"
	"github.com/jiva-studio/numen/modules/plugins/readers.epub/backend/epub/text"
	"github.com/jiva-studio/numen/modules/plugins/readers.epub/backend/epub/words"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// A book that names one part names its cover or its own title. Structure begins
// at two.
const MinimumParts = 2

// ReadParts reads the parts a book names, from whichever navigation
// document it carries. What counts is the entries that land in the text: a
// document whose entries do not is a document the book does not have, so the
// other one is read.
func ReadParts(files map[string]*zip.File, read opf.Package, extractor *text.Extractor) []text.Part {
	if raw, ok := archive.ReadEntry(files[read.NCX]); ok {
		if parts := extractor.GetParts(readNCXNavMap(raw, path.Dir(read.NCX))); len(parts) >= MinimumParts {
			return parts
		}
	}
	if raw, ok := archive.ReadEntry(files[read.Nav]); ok {
		return extractor.GetParts(readNavDocument(raw, path.Dir(read.Nav)))
	}
	return nil
}

// ReadPageEntries reads the pages of the printed book from the navigation control
// file's page list.
func ReadPageEntries(files map[string]*zip.File, read opf.Package) []text.Entry {
	raw, _ := archive.ReadEntry(files[read.NCX])
	return readNCXPageList(raw, path.Dir(read.NCX))
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
func readNCXNavMap(raw []byte, base string) []text.Entry {
	var document struct {
		Points []ncxPoint `xml:"navMap>navPoint"`
	}
	if err := archive.DecodeXML(raw, &document); err != nil {
		return nil
	}
	var out []text.Entry
	for _, point := range document.Points {
		out = appendPoint(out, point, base)
	}
	return out
}

// readNCXPageList reads the pages of the printed book. A page is named by its value
// when it has one; the label of a page target is often decoration around the
// number.
func readNCXPageList(raw []byte, base string) []text.Entry {
	var document struct {
		Targets []ncxPoint `xml:"pageList>pageTarget"`
	}
	if err := archive.DecodeXML(raw, &document); err != nil {
		return nil
	}
	var out []text.Entry
	for _, target := range document.Targets {
		if target.Content == nil {
			continue
		}
		name := target.Value
		if name == "" {
			name = target.Label
		}
		out = append(out, text.Entry{Title: name, Href: href.ResolveWithFragment(base, target.Content.Src)})
	}
	return out
}

func appendPoint(out []text.Entry, point ncxPoint, base string) []text.Entry {
	// A navPoint may have no content element, and then it points nowhere.
	if point.Content != nil {
		out = append(out, text.Entry{Title: point.Label, Href: href.ResolveWithFragment(base, point.Content.Src)})
	}
	for _, child := range point.Children {
		out = appendPoint(out, child, base)
	}
	return out
}

// readNavDocument reads an EPUB 3 navigation document: the links of its table of
// contents, in document order. A document with no table of contents is read for
// its links.
func readNavDocument(raw []byte, base string) []text.Entry {
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
		if words.HasToken(text.GetAttribute(n, "epub:type"), "toc") || words.HasToken(text.GetAttribute(n, "role"), "doc-toc") {
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

func appendLinks(out []text.Entry, n *html.Node, base string) []text.Entry {
	if n.DataAtom == atom.A {
		if target := text.GetAttribute(n, "href"); target != "" {
			out = append(out, text.Entry{Title: collectText(n), Href: href.ResolveWithFragment(base, target)})
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
