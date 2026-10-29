package epub

import (
	"fmt"
	"path"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/archive"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/markup"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/opf"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/text"
)

// Markup reads one spine document again, as elements and not as text.
//
// A document the book was not read from is not one to ask for: what a book holds
// beside its spine is not part of it.
func (b *Book) Markup(docPath string) (*Markup, error) {
	doc, ok := b.documents[docPath]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoDocument, docPath)
	}
	raw, _ := archive.ReadBounded(b.files[docPath], archive.MostPerDocument)
	nodes := text.ReadNodes(docPath, raw)

	// A spine document the manifest calls an SVG is a cover drawn around a
	// picture, and what a reader draws is that picture. One that draws none is
	// answered with the text it carries and nothing else.
	if b.mediaTypes[docPath] == opf.MediaSVG {
		if pos, ok := text.FindSVGImage(path.Dir(docPath), raw, func(name string) bool { return b.files[name] != nil }); ok {
			nodes = append([]markup.Node{{Name: "img", Text: "", Offset: 0, Attributes: []markup.Attribute{{Name: "src", Value: pos}}, Children: nil}}, nodes...)
		}
	}
	markup.Shift(nodes, doc.Offset)
	return &Markup{Path: docPath, Offset: doc.Offset, Length: doc.Length, Nodes: nodes}, nil
}
