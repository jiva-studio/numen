// Package text reads what a recognition, a transcript and a download wrote about a source, and
// says where in that text any offset is.
package text

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// A Document is a source, read.
type Document struct {
	// Text is what the source says, as one stream. Every offset below is a byte
	// offset into it.
	Text string

	// Parts are where the source names something, and are what chunks are cut
	// inside so that one never runs across a part into the next.
	Parts []domain.PartStart

	// named and paged are what the source calls the place an offset falls in,
	// in the vocabulary of its own format. Both ascend by offset, and either may
	// be empty: half the books read name neither.
	named []namedPlace
	paged []namedPlace
}

// A namedPlace is somewhere the source gives a name to.
type namedPlace struct {
	Offset int
	Name   string
}

// Locate is where an offset is: the part the source's own navigation, outline
// or headings name, and the page it falls on. Empty where the source has
// neither.
func (d *Document) Locate(offset int) string {
	named := make([]string, 0, 2)
	if i := findPrecedingIndex(d.named, offset); i >= 0 {
		named = append(named, d.named[i].Name)
	}
	if i := findPrecedingIndex(d.paged, offset); i >= 0 {
		named = append(named, d.paged[i].Name)
	}
	return strings.Join(named, ", ")
}

// GetNamesAt answers the parts that begin exactly at an offset: what a section
// starting here is called.
//
// A part and the first subsection inside it can begin at one place, and both
// name it. What is answered is every name, outermost first, so a question about
// either reaches the same place.
func (d *Document) GetNamesAt(offset int) []string {
	var names []string
	for _, m := range d.named {
		if m.Offset == offset {
			names = append(names, m.Name)
		}
		if m.Offset > offset {
			break
		}
	}
	return names
}

// page is what a page of a file is called: at its current position in it.
//
// is the one on the screen. What the paper printed is a second number for the
// same page, and a person shown both has to work out which is being talked
// about.
func page(pos int) string {
	return fmt.Sprintf("page %d of the file", pos+1)
}

func findPrecedingIndex(namedPlaces []namedPlace, offset int) int {
	return sort.Search(len(namedPlaces), func(i int) bool { return namedPlaces[i].Offset > offset }) - 1
}
