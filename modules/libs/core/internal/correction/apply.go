package correction

import (
	"cmp"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
)

// A change is one box put right: which box of the reading, the run of prose it
// covers, and what that run should say.
type change struct {
	index  int
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
func plan(boxes []highlight.Box, lines []Line) walk {
	named := make([]Line, 0, len(lines))
	for _, line := range lines {
		if line.Number < 0 || line.Number >= len(boxes) {
			continue
		}
		named = append(named, line)
	}
	if len(named) == 0 {
		return walk{grown: []int{0}}
	}

	byNumber := func(a, b Line) int { return cmp.Compare(a.Number, b.Number) }
	if !slices.IsSortedFunc(named, byNumber) {
		slices.SortStableFunc(named, byNumber)
	}

	w := walk{changes: make([]change, 0, len(named)), grown: make([]int, 1, len(named)+1)}
	end := 0
	for k, line := range named {
		if k+1 < len(named) && named[k+1].Number == line.Number {
			continue
		}
		box := boxes[line.Number]
		if box.From < end || box.To < box.From {
			continue
		}
		one := change{index: line.Number, start: box.From, length: box.Len(), text: line.Text}
		w.changes = append(w.changes, one)
		w.grown = append(w.grown, w.grown[len(w.grown)-1]+one.delta())
		end = box.To
	}
	return w
}

// getStartFrom is the position of the first correction whose box begins at or
// after offset o.
func (w walk) getStartFrom(o int) int {
	low, high := 0, len(w.changes)
	for low < high {
		mid := int(uint(low+high) >> 1)
		if w.changes[mid].start < o {
			low = mid + 1
		} else {
			high = mid
		}
	}
	return low
}

// getGrowthBefore is the growth of the prose before offset o: the corrections
// whose box begins earlier.
func (w walk) getGrowthBefore(o int) int {
	return w.grown[w.getStartFrom(o)]
}

// getGrowthInside is the growth of the corrections lying wholly within the run.
func (w walk) getGrowthInside(start, length int) int {
	end := start + length
	from := w.getStartFrom(start)
	to, high := 0, len(w.changes)
	for to < high {
		mid := int(uint(to+high) >> 1)
		if w.changes[mid].start+w.changes[mid].length > end {
			high = mid
		} else {
			to = mid + 1
		}
	}
	if to < from {
		to = from
	}
	return w.grown[to] - w.grown[from]
}

// Boxes are where the runs of a reading sit once its corrections are in it.
func Boxes(boxes []highlight.Box, lines []Line) []highlight.Box {
	w := plan(boxes, lines)
	if len(w.changes) == 0 {
		return boxes
	}
	out := make([]highlight.Box, len(boxes))
	copy(out, boxes)
	next := 0
	for i := range out {
		covers := out[i].Len()
		out[i].From += w.grown[next]
		if next < len(w.changes) && w.changes[next].index == i {
			covers = len(w.changes[next].text)
			next++
		}
		out[i].To = out[i].From + covers
	}
	return out
}

// Prose is a reading's text with its corrections in it, and the pages and the
// parts where they now stand.
func Prose(prose string, marks []ocr.PageStart, boxes []highlight.Box, parts []ocr.Part, lines []Line) (string, []ocr.PageStart, []ocr.Part) {
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
		start, end := clamp(one.start, cursor, len(prose)), clamp(one.start+one.length, cursor, len(prose))
		out.WriteString(prose[cursor:start])
		out.WriteString(one.text)
		cursor = end
	}
	out.WriteString(prose[cursor:])

	pages := marks
	if len(marks) > 0 {
		pages = make([]ocr.PageStart, len(marks))
		for i, mark := range marks {
			pages[i] = ocr.PageStart{Offset: mark.Offset + w.getGrowthBefore(mark.Offset)}
		}
	}
	named := parts
	if len(parts) > 0 {
		named = make([]ocr.Part, len(parts))
		for i, part := range parts {
			named[i] = ocr.Part{
				Start:  part.Start + w.getGrowthBefore(part.Start),
				Length: part.Length + w.getGrowthInside(part.Start, part.Length),
				Depth:  part.Depth,
			}
		}
	}
	return out.String(), pages, named
}

// clamp is offset o held to the run [low, high].
func clamp(o, low, high int) int {
	if o < low {
		return low
	}
	if o > high {
		return high
	}
	return o
}
