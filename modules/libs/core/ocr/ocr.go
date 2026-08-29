// Package ocr turns what a model saw on a page into the prose the page prints.
//
// It is pure: no filesystem, no clock, no database, and no model. What divides
// a page into regions and what reads a line are given to it as their answers,
// and what it does is put those answers back into the order a person reads.
//
// The vocabulary is the page's: a region is a part of a page, a line is a run
// of words the detector found, and a place in the result is a printed page
// number. Another kind of source would not share any of them.
package ocr

import (
	"image"
	"regexp"
	"sort"
	"strings"
)

// A Region is one part of a page: what it is, where it is, and where it comes
// in the order the page is read.
type Region struct {
	Label string
	Score float32
	Rect  image.Rectangle
	// Order is the place the page's own reading order gives it. Sorting on it
	// is the whole of what would otherwise be a cut of the page into columns.
	Order int
}

// A Line is a run of words a detector found, and what a recogniser read in it.
type Line struct {
	Box   image.Rectangle
	Text  string
	Score float32
}

// A Span is one box of a region, in what was written from it: the rectangle it
// covers on the page, and the run of bytes it produced.
type Span struct {
	Box    image.Rectangle
	Start  int
	Length int
}

// A Block is one region of a page, written out.
type Block struct {
	Label string
	Text  string
	// Head says the region is a heading, and Depth is how far in the part it
	// opens sits. A document title stands above the section titles within it.
	Head  bool
	Depth int
	// Spans are where on the page each run of Text was read. A recogniser that
	// reports no rectangles leaves them empty.
	Spans []Span
}

// A Page is one page of a document, read.
//
// It carries no name of its own. What a page is called is where it stands in
// the document, and that is what a person is shown and what a viewer opens at.
type Page struct {
	// At is which page of the document this is, counted from zero.
	At int
	// Size is the page as it was rendered, which the rectangles are addressed
	// from. A zero size is a page nothing was measured on.
	Size image.Point
	// Blocks are what it says, in the order it is read.
	Blocks []Block
}

// Distinct drops a region that covers one already kept.
//
// Detection returns a region more than once, and returns a region inside
// another. Every region is read on its own, so a region kept twice is a
// paragraph read twice, and a region inside another is a paragraph read once as
// itself and once as part of its neighbour.
//
// The one kept is the one the model was surest of, so the regions are ordered by
// score first and put back in reading order after.
func Distinct(regions []Region, most float64) []Region {
	byScore := append([]Region(nil), regions...)
	sort.SliceStable(byScore, func(a, b int) bool { return byScore[a].Score > byScore[b].Score })

	kept := make([]Region, 0, len(byScore))
	for _, r := range byScore {
		covered := false
		for _, k := range kept {
			if overlap(r.Rect, k.Rect) > most || inside(r.Rect, k.Rect) {
				covered = true
				break
			}
		}
		if !covered {
			kept = append(kept, r)
		}
	}
	sort.SliceStable(kept, func(a, b int) bool { return kept[a].Order < kept[b].Order })
	return kept
}

// overlap is the share of the two rectangles together that both cover.
func overlap(a, b image.Rectangle) float64 {
	both := a.Intersect(b)
	if both.Empty() {
		return 0
	}
	common := float64(both.Dx() * both.Dy())
	either := float64(a.Dx()*a.Dy()) + float64(b.Dx()*b.Dy()) - common
	if either <= 0 {
		return 0
	}
	return common / either
}

// inside says whether almost all of one rectangle is within the other.
func inside(a, b image.Rectangle) bool {
	both := a.Intersect(b)
	if both.Empty() {
		return false
	}
	return float64(both.Dx()*both.Dy()) > 0.9*float64(a.Dx()*a.Dy())
}

// Assemble writes one region out as running prose, and says where on the page
// each run of it was read.
func Assemble(lines []Line) (string, []Span) {
	var out strings.Builder
	var spans []Span
	for _, line := range group(lines) {
		text, boxes := written(line)
		if text == "" {
			continue
		}
		joined := out.String()
		if hyphen.MatchString(joined) {
			out.Reset()
			out.WriteString(hyphen.ReplaceAllString(joined, "$1"))
			// The hyphen is gone from the end of the box that carried it, so
			// that box covers one byte fewer than it wrote.
			if n := len(spans); n > 0 {
				spans[n-1].Length -= len(joined) - out.Len()
			}
		} else if joined != "" {
			out.WriteString(" ")
		}
		at := out.Len()
		for _, span := range boxes {
			span.Start += at
			spans = append(spans, span)
		}
		out.WriteString(text)
	}
	return out.String(), spans
}

// group divides a region's lines into the lines the page prints.
//
// Two boxes belong to one line when they overlap vertically by most of their
// height, which holds for a line sitting slightly askew on the scan and does not
// hold for the line beneath it.
func group(lines []Line) [][]Line {
	sorted := append([]Line(nil), lines...)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].Box.Min.Y < sorted[b].Box.Min.Y })

	var out [][]Line
	var current []Line
	for _, line := range sorted {
		if len(current) > 0 && shared(current, line) < 0.5 {
			out = append(out, ordered(current))
			current = nil
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		out = append(out, ordered(current))
	}
	return out
}

func ordered(line []Line) []Line {
	sort.SliceStable(line, func(a, b int) bool { return line[a].Box.Min.X < line[b].Box.Min.X })
	return line
}

// shared is how much of the shorter of the two a line and a box have in common
// vertically.
func shared(line []Line, box Line) float64 {
	top, bottom := box.Box.Min.Y, box.Box.Max.Y
	height := box.Box.Dy()
	for _, l := range line {
		top = max(top, l.Box.Min.Y)
		bottom = min(bottom, l.Box.Max.Y)
		height = min(height, l.Box.Dy())
	}
	if height <= 0 {
		return 0
	}
	return float64(max(bottom-top, 0)) / float64(height)
}

// written is one line of the page, and where in it each box was read. The
// offsets are in the line's own text.
//
// Detection cuts a line where the printing leaves a gap, so two boxes of one
// line are two words. The gap itself is not measurable here: a detector widens
// every box it returns by a fixed number of pixels, and neighbours therefore
// overlap however far apart the words were.
func written(line []Line) (string, []Span) {
	var out strings.Builder
	var spans []Span
	for _, box := range line {
		// A box the recogniser read as nothing is not a word.
		text := strings.TrimSpace(box.Text)
		if text == "" {
			continue
		}
		if out.Len() > 0 {
			out.WriteString(" ")
		}
		spans = append(spans, Span{Box: box.Box, Start: out.Len(), Length: len(text)})
		out.WriteString(text)
	}
	return out.String(), spans
}

// A line broken by a hyphen continues in the next one.
var hyphen = regexp.MustCompile(`(\pL)[-‐‑\x{00ad}]$`)
