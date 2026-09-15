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

// GetBoxesOver is the boxes covering a run of the prose, in the order they were
// read. A run standing nowhere is covered by none.
//
// The boxes are in the order they were read, so the run is found by halving and
// then walked to its end. A run crossing a page carries boxes from both of them.
func GetBoxesOver(boxes []Box, span domain.Span) []Box {
	if span.Empty() || len(boxes) == 0 {
		return nil
	}

	// The first box that reaches into the run. A box before it ends before the
	// run begins.
	at := sort.Search(len(boxes), func(i int) bool {
		return boxes[i].To > span.From
	})

	var out []Box
	for ; at < len(boxes) && boxes[at].From < span.To; at++ {
		out = append(out, boxes[at])
	}
	return out
}
