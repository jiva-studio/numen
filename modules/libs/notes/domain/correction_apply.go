package domain

import (
	"cmp"
	"slices"
	"strings"
)

// A correctionChange is one box put right: which box of the recognition, the run of prose it
// covers, and what that run should say.
type correctionChange struct {
	index  int
	start  int
	length int
	text   string
}

// delta is how much the prose grows for this correction.
func (c correctionChange) delta() int { return len(c.text) - c.length }

// A correctionWalk is a recognition's corrections in reading order, with the growth of the
// prose accumulated across them. Where the boxes land and where the prose, the
// marks and the parts land are all read off it.
type correctionWalk struct {
	changes []correctionChange
	// grown[i] is the growth of the first i corrections, so grown[0] is zero.
	grown []int
}

// plan is the corrections a recognition's boxes answer to, in reading order. A
// number no box answers to is dropped, a line named twice keeps what came last,
// and a box reaching back into the one before it is left as it was.
func planCorrections(boxes []HighlightBox, lines []CorrectionLine) correctionWalk {
	named := make([]CorrectionLine, 0, len(lines))
	for _, line := range lines {
		if line.Number < 0 || line.Number >= len(boxes) {
			continue
		}
		named = append(named, line)
	}
	if len(named) == 0 {
		return correctionWalk{grown: []int{0}}
	}

	byNumber := func(a, b CorrectionLine) int { return cmp.Compare(a.Number, b.Number) }
	if !slices.IsSortedFunc(named, byNumber) {
		slices.SortStableFunc(named, byNumber)
	}

	w := correctionWalk{changes: make([]correctionChange, 0, len(named)), grown: make([]int, 1, len(named)+1)}
	end := 0
	for k, line := range named {
		if k+1 < len(named) && named[k+1].Number == line.Number {
			continue
		}
		box := boxes[line.Number]
		if box.From < end || box.To < box.From {
			continue
		}
		one := correctionChange{index: line.Number, start: box.From, length: box.Len(), text: line.Text}
		w.changes = append(w.changes, one)
		w.grown = append(w.grown, w.grown[len(w.grown)-1]+one.delta())
		end = box.To
	}
	return w
}

// getStartFrom is the position of the first correction whose box begins at or
// after offset o.
func (w correctionWalk) getStartFrom(o int) int {
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
func (w correctionWalk) getGrowthBefore(o int) int {
	return w.grown[w.getStartFrom(o)]
}

// getGrowthInside is the growth of the corrections lying wholly within the run.
func (w correctionWalk) getGrowthInside(start, length int) int {
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

// GetCorrectedBoxes returns where the runs of a recognition sit once its corrections are in it.
func GetCorrectedBoxes(boxes []HighlightBox, lines []CorrectionLine) []HighlightBox {
	w := planCorrections(boxes, lines)
	if len(w.changes) == 0 {
		return boxes
	}
	out := make([]HighlightBox, len(boxes))
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

// GetCorrectedProse returns a recognition's text with its corrections in it, and the pages and the
// parts where they now stand.
func GetCorrectedProse(prose string, marks []OCRPageStart, boxes []HighlightBox, parts []OCRPart, lines []CorrectionLine) (string, []OCRPageStart, []OCRPart) {
	w := planCorrections(boxes, lines)
	if len(w.changes) == 0 {
		return prose, marks, parts
	}

	// A correction covers the run its box names. A box reaching past the text
	// it was read from takes what there is of it.
	var out strings.Builder
	out.Grow(len(prose) + w.grown[len(w.grown)-1])
	cursor := 0
	for _, one := range w.changes {
		start, end := clampOffset(one.start, cursor, len(prose)), clampOffset(one.start+one.length, cursor, len(prose))
		out.WriteString(prose[cursor:start])
		out.WriteString(one.text)
		cursor = end
	}
	out.WriteString(prose[cursor:])

	pages := marks
	if len(marks) > 0 {
		pages = make([]OCRPageStart, len(marks))
		for i, mark := range marks {
			pages[i] = OCRPageStart{Offset: mark.Offset + w.getGrowthBefore(mark.Offset)}
		}
	}
	named := parts
	if len(parts) > 0 {
		named = make([]OCRPart, len(parts))
		for i, part := range parts {
			named[i] = OCRPart{
				Start:  part.Start + w.getGrowthBefore(part.Start),
				Length: part.Length + w.getGrowthInside(part.Start, part.Length),
				Depth:  part.Depth,
			}
		}
	}
	return out.String(), pages, named
}

func clampOffset(o, low, high int) int {
	if o < low {
		return low
	}
	if o > high {
		return high
	}
	return o
}
