package epub

import (
	"slices"
	"sort"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/archive"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/cfi"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/text"
)

// Places are where the block elements of a book begin in its Text, and the text nodes its Text was
// written from, which a place in the Text and its EPUB CFI are found from.
type Places struct {
	// Blocks are where each block element begins in the book's Text, ascending.
	Blocks []int
	runs   []placedRun
}

// A placedRun is a text node of a spine document, the CFI steps to the document, and where the
// document begins in the book's Text.
type placedRun struct {
	text.Run
	pkg    []int
	offset int
}

func (r placedRun) end() int { return r.offset + r.End }

// ReadPlaces reads every spine document again for its places.
func (b *Book) ReadPlaces() Places {
	return b.readPlaces(b.Documents)
}

func (b *Book) readPlaces(docs []Document) Places {
	var places Places
	for _, doc := range docs {
		raw, ok := archive.ReadBounded(b.files[doc.Path], archive.MostPerDocument)
		if !ok {
			continue
		}
		pkg := b.getPackageSteps(doc)
		read := text.ReadPlaces(doc.Path, raw)
		for _, block := range read.Blocks {
			places.Blocks = append(places.Blocks, doc.Offset+block)
		}
		for _, run := range read.Runs {
			places.runs = append(places.runs, placedRun{Run: run, pkg: pkg, offset: doc.Offset})
		}
	}
	return places
}

// getPackageSteps is the CFI steps through the package document to a document's itemref.
func (b *Book) getPackageSteps(doc Document) []int {
	return []int{b.spineStep, b.itemSteps[doc.Spine]}
}

// GetCFI is the CFI of the character at an offset of the Text: the first character written at or
// after it, or the end of the last text node where none is. It is false for a book with no text
// node.
func (p Places) GetCFI(offset int) (cfi.CFI, bool) {
	if len(p.runs) == 0 {
		return cfi.CFI{Package: nil, Document: nil, Offset: 0}, false
	}
	i := sort.Search(len(p.runs), func(i int) bool { return p.runs[i].end() > offset })
	if i == len(p.runs) {
		last := p.runs[len(p.runs)-1]
		return cfi.CFI{Package: last.pkg, Document: last.Steps, Offset: last.Base + last.Length()}, true
	}
	run := p.runs[i]
	return cfi.CFI{Package: run.pkg, Document: run.Steps, Offset: run.GetCharOffset(offset - run.offset)}, true
}

// GetCFI is the CFI of the character at an offset of the Text, as Places.GetCFI finds it, reading
// only the documents it needs.
func (b *Book) GetCFI(offset int) (cfi.CFI, bool) {
	from := findPreceding(len(b.Documents), offset, func(i int) int { return b.Documents[i].Offset })
	for i := max(from, 0); i < len(b.Documents); i++ {
		places := b.readPlaces(b.Documents[i : i+1])
		if len(places.runs) > 0 && places.runs[len(places.runs)-1].end() > offset {
			return places.GetCFI(offset)
		}
	}
	return b.ReadPlaces().GetCFI(offset)
}

// FindOffset is the offset of the Text a CFI names. A CFI naming an element is where the element's
// first text begins, and one naming a node that wrote no text is where the next text begins. It is
// false for a CFI naming no document of the spine.
func (b *Book) FindOffset(place cfi.CFI) (int, bool) {
	for _, doc := range b.Documents {
		if !slices.Equal(b.getPackageSteps(doc), place.Package) {
			continue
		}
		return findDocumentOffset(doc, b.readPlaces([]Document{doc}).runs, place), true
	}
	return 0, false
}

// findDocumentOffset is the offset a CFI names among the runs of one document.
func findDocumentOffset(doc Document, runs []placedRun, place cfi.CFI) int {
	i := sort.Search(len(runs), func(i int) bool { return slices.Compare(runs[i].Steps, place.Document) >= 0 })
	if i == len(runs) {
		return doc.Offset + doc.Length
	}
	if !slices.Equal(runs[i].Steps, place.Document) || place.Offset < 0 {
		return runs[i].offset + runs[i].Start
	}
	// A comment splits one run of character data into nodes with the same steps.
	for i+1 < len(runs) && slices.Equal(runs[i+1].Steps, place.Document) && runs[i+1].Base <= place.Offset {
		i++
	}
	return runs[i].offset + runs[i].FindOffset(place.Offset)
}
