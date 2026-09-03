package fixes

import (
	"sort"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/lit"
	"github.com/jiva-studio/numen/modules/libs/core/ocr"
)

// A change is one box put right: which box of the reading, the run of prose it
// covers, and what that run should say.
type change struct {
	at     int
	start  int
	length int
	text   string
}

// delta is how much the prose grows for this correction.
func (c change) delta() int { return len(c.text) - c.length }

// A walk is a reading's corrections in reading order, with the growth of the
// prose accumulated across them. Where the boxes land and where the prose, the
// marks and the parts land are all read off it.
type walk struct {
	changes []change
	// grown[i] is the growth of the first i corrections, so grown[0] is zero.
	grown []int
}

// plan is the corrections a reading's boxes answer to, in reading order. A
// number no box answers to is dropped, a line named twice keeps what came last,
// and a box reaching back into the one before it is left as it was.
func plan(boxes []lit.Box, lines []Line) walk {
	w := walk{grown: []int{0}}
	said := make(map[int]string, len(lines))
	for _, line := range lines {
		if line.At < 0 || line.At >= len(boxes) {
			continue
		}
		said[line.At] = line.Text
	}
	if len(said) == 0 {
		return w
	}

	at := make([]int, 0, len(said))
	for i := range said {
		at = append(at, i)
	}
	sort.Ints(at)

	end := 0
	for _, i := range at {
		box := boxes[i]
		if box.Start < end || box.Length < 0 {
			continue
		}
		one := change{at: i, start: box.Start, length: box.Length, text: said[i]}
		w.changes = append(w.changes, one)
		w.grown = append(w.grown, w.grown[len(w.grown)-1]+one.delta())
		end = box.Start + box.Length
	}
	return w
}

// before is the growth of the prose before offset o: the corrections whose box
// begins earlier.
func (w walk) before(o int) int {
	n := sort.Search(len(w.changes), func(i int) bool { return w.changes[i].start >= o })
	return w.grown[n]
}

// inside is the growth of the corrections lying wholly within the run.
func (w walk) inside(start, length int) int {
	end := start + length
	from := sort.Search(len(w.changes), func(i int) bool { return w.changes[i].start >= start })
	to := sort.Search(len(w.changes), func(i int) bool { return w.changes[i].start+w.changes[i].length > end })
	if to < from {
		to = from
	}
	return w.grown[to] - w.grown[from]
}

// Boxes are where the runs of a reading sit once its corrections are in it.
func Boxes(boxes []lit.Box, lines []Line) []lit.Box {
	w := plan(boxes, lines)
	if len(w.changes) == 0 {
		return boxes
	}
	out := make([]lit.Box, len(boxes))
	copy(out, boxes)
	next := 0
	for i := range out {
		out[i].Start += w.grown[next]
		if next < len(w.changes) && w.changes[next].at == i {
			out[i].Length = len(w.changes[next].text)
			next++
		}
	}
	return out
}

// Prose is a reading's text with its corrections in it, and the pages and the
// parts where they now stand.
func Prose(prose string, marks []ocr.PageStart, boxes []lit.Box, parts []ocr.Part, lines []Line) (string, []ocr.PageStart, []ocr.Part) {
	w := plan(boxes, lines)
	if len(w.changes) == 0 {
		return prose, marks, parts
	}

	// A correction covers the run its box names. A box reaching past the text
	// it was read from takes what there is of it.
	var out strings.Builder
	out.Grow(len(prose) + w.grown[len(w.grown)-1])
	cursor := 0
	for _, one := range w.changes {
		start, end := within(one.start, cursor, len(prose)), within(one.start+one.length, cursor, len(prose))
		out.WriteString(prose[cursor:start])
		out.WriteString(one.text)
		cursor = end
	}
	out.WriteString(prose[cursor:])

	pages := marks
	if len(marks) > 0 {
		pages = make([]ocr.PageStart, len(marks))
		for i, mark := range marks {
			pages[i] = ocr.PageStart{Offset: mark.Offset + w.before(mark.Offset)}
		}
	}
	named := parts
	if len(parts) > 0 {
		named = make([]ocr.Part, len(parts))
		for i, part := range parts {
			named[i] = ocr.Part{
				Start:  part.Start + w.before(part.Start),
				Length: part.Length + w.inside(part.Start, part.Length),
				Depth:  part.Depth,
			}
		}
	}
	return out.String(), pages, named
}

// within is offset o held to the run [low, high].
func within(o, low, high int) int {
	if o < low {
		return low
	}
	if o > high {
		return high
	}
	return o
}
