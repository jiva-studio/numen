package engine

import (
	"cmp"
	"image"
	"iter"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Region is one part of a page: what it is, where it is, and where it comes
// in the order the page is read.
type Region struct {
	Label string
	Score float32
	Rect  image.Rectangle
	Order int
}

// Line is a run of words a detector found, and what a recogniser read in it.
type Line struct {
	Box   image.Rectangle
	Text  string
	Score float32
}

// ByteRange represents a byte offset range.
type ByteRange struct {
	Start int
	End   int
}

// Box is one box of a region: the rectangle it covers on the page, and the run of bytes it produced.
type Box struct {
	Rect  image.Rectangle
	Range ByteRange
}

// Block is one region of a page, written out.
type Block struct {
	Label     string
	Text      string
	IsHeading bool
	Depth     int
	Boxes     []Box
}

// Distinct drops a region that covers one already kept.
func Distinct(regions []Region, most float64) []Region {
	slices.SortStableFunc(regions, func(a, b Region) int {
		switch {
		case a.Score > b.Score:
			return -1
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY,CONDITIONALS_NEGATION reason="slices.SortStableFunc tests only for negative comparison results; returning 1 or 0 produces the same order"
		case b.Score > a.Score:
			return 1
		}
		return 0
	})

	result := regions[:0]
	for _, r := range regions {
		covered := false
		for _, k := range result {
			if overlap(r.Rect, k.Rect) > most || isInside(r.Rect, k.Rect) {
				covered = true
				// gomutants:disable-next-line INVERT_LOOP_CTRL reason="covered is already true; continuing the loop checks further kept regions with identical results"
				break
			}
		}
		if !covered {
			result = append(result, r)
		}
	}
	slices.SortStableFunc(result, func(a, b Region) int { return cmp.Compare(a.Order, b.Order) })
	return result
}

func overlap(a, b image.Rectangle) float64 {
	both := a.Intersect(b)
	if both.Empty() {
		return 0
	}
	common := float64(both.Dx() * both.Dy())
	either := float64(a.Dx()*a.Dy()) + float64(b.Dx()*b.Dy()) - common
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_DECREMENT,BRANCH_IF reason="either is area(a)+area(b)-common; since common is positive, either is always positive"
	if either <= 0 {
		return 0
	}
	return common / either
}

func isInside(a, b image.Rectangle) bool {
	both := a.Intersect(b)
	if both.Empty() {
		return false
	}
	return float64(both.Dx()*both.Dy()) > 0.9*float64(a.Dx()*a.Dy())
}

// Assemble writes one region out as running prose, and says where on the page
// each run of it was read.
func Assemble(lines []Line) (string, []Box) {
	size := len(lines)
	// gomutants:disable-next-line RANGE_BREAK reason="size is a capacity hint for make; the assembled text and spans are identical at any capacity"
	for _, one := range lines {
		// gomutants:disable-next-line REMOVE_SELF_ASSIGNMENTS reason="size is a capacity hint for make; the assembled text and spans are identical at any capacity"
		size += len(one.Text)
	}
	out := make([]byte, 0, size)
	assembledBoxes := make([]Box, 0, len(lines))
	for line := range group(lines) {
		text, boxes := writeLine(line)
		if text == "" {
			continue
		}
		if n := getHyphenLength(out); n > 0 {
			out = out[:len(out)-n]
			// gomutants:disable-next-line INTEGER_DECREMENT reason="len(kept) is at least 1 whenever out holds text"
			if k := len(assembledBoxes); k > 0 {
				assembledBoxes[k-1].Range.End -= n
			}
		} else if len(out) > 0 {
			out = append(out, ' ')
		}
		pos := len(out)
		for _, box := range boxes {
			box.Range.Start += pos
			box.Range.End += pos
			assembledBoxes = append(assembledBoxes, box)
		}
		out = append(out, text...)
	}
	return string(out), assembledBoxes
}

func getHyphenLength(text []byte) int {
	mark, width := utf8.DecodeLastRune(text)
	if !isHyphen(mark) {
		return 0
	}
	letter, _ := utf8.DecodeLastRune(text[:len(text)-width])
	if !unicode.IsLetter(letter) {
		return 0
	}
	return width
}

func isHyphen(r rune) bool {
	switch r {
	case '-', '\u2010', '\u2011', '\u00ad':
		return true
	}
	return false
}

func group(lines []Line) iter.Seq[[]Line] {
	sorted := slices.Clone(lines)
	slices.SortStableFunc(sorted, func(a, b Line) int { return cmp.Compare(a.Box.Min.Y, b.Box.Min.Y) })

	return func(yield func([]Line) bool) {
		start := 0
		for i, line := range sorted {
			if i > start && measureVerticalOverlap(sorted[start:i], line) < 0.5 {
				if !yield(sortLine(sorted[start:i])) {
					return
				}
				start = i
			}
		}
		if start < len(sorted) {
			yield(sortLine(sorted[start:]))
		}
	}
}

func sortLine(line []Line) []Line {
	slices.SortStableFunc(line, func(a, b Line) int { return cmp.Compare(a.Box.Min.X, b.Box.Min.X) })
	return line
}

func measureVerticalOverlap(line []Line, box Line) float64 {
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

func writeLine(line []Line) (string, []Box) {
	size := len(line)
	// gomutants:disable-next-line RANGE_BREAK reason="size is a capacity hint for Grow; the assembled text is identical at any capacity"
	for _, one := range line {
		// gomutants:disable-next-line REMOVE_SELF_ASSIGNMENTS reason="size is a capacity hint for Grow; the assembled text is identical at any capacity"
		size += len(one.Text)
	}
	var out strings.Builder
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Grow sets capacity only; strings.Builder grows dynamically when omitted"
	out.Grow(size)
	lineBoxes := make([]Box, 0, len(line))
	for _, one := range line {
		text := strings.TrimSpace(one.Text)
		if text == "" {
			continue
		}
		if out.Len() > 0 {
			out.WriteString(" ")
		}
		pos := out.Len()
		lineBoxes = append(lineBoxes, Box{Rect: one.Box, Range: ByteRange{Start: pos, End: pos + len(text)}})
		out.WriteString(text)
	}
	return out.String(), lineBoxes
}
