package text

import (
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/domain/epub/href"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/domain/epub/words"
)

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
		out = append(out, Part{Title: title, Offset: offset, Level: 0})
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
