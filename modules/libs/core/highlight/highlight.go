// Package highlight says where a run of a source's text falls on the page it
// was read from.
//
// It is pure: no filesystem, no clock, no model. Two things produce what is
// here — a model reading a scan, and a document's own text layer — and nothing
// above this point asks which. A viewer asks where a run of the prose is and is
// told.
//
// The rectangle is a fraction of the page, so a page drawn at any size lines up
// by multiplying and nothing is recomputed.
package highlight

import (
	"sort"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A Box is one run of prose and where it was read: the page it is on, the run
// of bytes in the text, and the rectangle it covers.
type Box struct {
	Page int
	domain.Span
	Rect
}

// A Rect is a place on a page, in fractions of it.
type Rect struct {
	MinX, MinY, MaxX, MaxY float32
}

// A Page is one page and what to light on it.
type Page struct {
	Index int
	Rects []Rect
}

// Pages is where a run of the prose sits: the pages it falls on and, on each,
// the rectangles covering it.
//
// The boxes are in the order they were read, so the run is found by halving and
// then walked to its end. A run crossing a page is on both of them.
func Pages(boxes []Box, span domain.Span) []Page {
	if span.Empty() || len(boxes) == 0 {
		return nil
	}

	// The first box that reaches into the run. A box before it ends before the
	// run begins.
	at := sort.Search(len(boxes), func(i int) bool {
		return boxes[i].To > span.From
	})

	var out []Page
	for ; at < len(boxes) && boxes[at].From < span.To; at++ {
		box := boxes[at]
		if n := len(out); n > 0 && out[n-1].Index == box.Page {
			out[n-1].Rects = append(out[n-1].Rects, box.Rect)
			continue
		}
		out = append(out, Page{Index: box.Page, Rects: []Rect{box.Rect}})
	}
	return out
}
