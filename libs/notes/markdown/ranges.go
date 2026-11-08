package markdown

import (
	"strings"
	"unicode/utf8"
)

// Range is a run of prose, given by its start and end as byte offsets into the text
// it was found in. A client counts in other units (domain.UnitRange); CountUTF16
// converts, and no offset crosses that boundary unconverted.
type Range struct {
	Start int
	End   int
}

// FindRanges returns the spans of text that match wanted: first exactly, then with
// punctuation and spacing normalized, where a run of spacing in a line is one space.
// An exact match is never reported as approximate.
// Offsets are into text as given, so the replaced bytes are the ones the user wrote.
// isApproximate is true when a span was found only by the normalized reading.
func FindRanges(text, wanted string) (spans []Range, isApproximate bool) {
	if wanted == "" {
		return nil, false
	}
	if found := occurrences(text, wanted); len(found) > 0 {
		return found, false
	}
	for _, read := range []func(string) mappedText{foldPunctuation, foldSpacing} {
		target, sought := read(text), read(wanted)
		// gomutants:disable-next-line BRANCH_IF reason="foldPunctuation and foldSpacing preserve at least one rune for any non-empty wanted"
		if sought.text == "" {
			// gomutants:disable-next-line INVERT_LOOP_CTRL reason="sought.text is never empty when wanted is non-empty"
			continue
		}
		found := occurrences(target.text, sought.text)
		if len(found) == 0 {
			continue
		}
		foundSpans := make([]Range, 0, len(found))
		for _, span := range found {
			foundSpans = append(foundSpans, Range{Start: target.offsets[span.Start], End: target.offsets[span.End]})
		}
		return foundSpans, true
	}
	return nil, false
}

func occurrences(text, wanted string) []Range {
	var foundSpans []Range
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="at from == len(text) strings.Index on an empty slice with non-empty wanted returns -1"
	for from := 0; from <= len(text); {
		next := strings.Index(text[from:], wanted)
		if next < 0 {
			return foundSpans
		}
		foundSpans = append(foundSpans, Range{Start: from + next, End: from + next + len(wanted)})
		from += next + len(wanted)
	}
	return foundSpans
}

// came from. There is one more offset than there are bytes, so the end of a
// span reads back as well as its beginning.
type mappedText struct {
	text    string
	offsets []int
}

func foldPunctuation(text string) mappedText {
	var out strings.Builder
	// gomutants:disable-next-line ARITHMETIC_BASE,INTEGER_DECREMENT,INTEGER_INCREMENT reason="make capacity hint does not alter the appended slice contents"
	offsets := make([]int, 0, len(text)+1)
	for from, r := range text {
		before := out.Len()
		out.WriteRune(getPlainRune(r))
		for range out.Len() - before {
			offsets = append(offsets, from)
		}
	}
	return mappedText{text: out.String(), offsets: append(offsets, len(text))}
}

func foldSpacing(text string) mappedText {
	var out strings.Builder
	// gomutants:disable-next-line ARITHMETIC_BASE,INTEGER_DECREMENT,INTEGER_INCREMENT reason="make capacity hint does not alter the appended slice contents"
	offsets := make([]int, 0, len(text)+1)
	spacing := false
	for from, r := range text {
		mark := getPlainRune(r)
		switch {
		case mark == '\n':
			spacing = false
		case mark == ' ' || mark == '\t':
			if spacing {
				continue
			}
			mark, spacing = ' ', true
		default:
			spacing = false
		}
		before := out.Len()
		out.WriteRune(mark)
		for range out.Len() - before {
			offsets = append(offsets, from)
		}
	}
	return mappedText{text: out.String(), offsets: append(offsets, len(text))}
}

func getPlainRune(r rune) rune {
	switch {
	// Hyphens and dashes, U+2010 to U+2015, and the minus sign U+2212.
	case r >= '‐' && r <= '―', r == '−':
		return '-'
	// Single quotes, U+2018 to U+201B, and the angled pair U+2039 and U+203A.
	case r >= '‘' && r <= '‛', r == '‹', r == '›':
		return '\''
	// Double quotes, U+201C to U+201F, and the angled pair U+00AB and U+00BB.
	case r >= '“' && r <= '‟', r == '«', r == '»':
		return '"'
	// Spaces that are not the space bar: U+00A0, U+2002 to U+200A, U+202F,
	// U+205F and U+3000.
	case r == ' ', r >= ' ' && r <= ' ', r == ' ',
		r == ' ', r == '　':
		return ' '
	}
	return r
}

// Diff is the span of `was` that `now` does not have, and the replacement text.
//
// The common prefix and suffix are excluded, so replacing one note's whole prose
// with another names only the sentence that changed. The span is widened to whole
// words, and two identical texts yield no span.
func Diff(was, now string) (Range, string) {
	if was == now {
		return Range{Start: len(was), End: len(was)}, ""
	}

	head := 0
	for head < len(was) && head < len(now) && was[head] == now[head] {
		head++
	}
	for head > 0 && !canSpanBeginAt(was, head) {
		head--
	}

	most := min(len(was), len(now)) - head
	tail := 0
	for tail < most && was[len(was)-tail-1] == now[len(now)-tail-1] {
		tail++
	}
	for tail > 0 && !canSpanEndAt(was, len(was)-tail) {
		tail--
	}

	return Range{Start: head, End: len(was) - tail}, now[head : len(now)-tail]
}

func canSpanBeginAt(text string, offset int) bool {
	if offset <= 0 {
		return true
	}
	if offset < len(text) && !utf8.RuneStart(text[offset]) {
		return false
	}
	return isSpace(text[offset-1])
}

func canSpanEndAt(text string, offset int) bool {
	if offset >= len(text) {
		return true
	}
	if !utf8.RuneStart(text[offset]) {
		return false
	}
	return isSpace(text[offset])
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// CountUTF16 is `at`, a byte offset into text, as a client counts text: in
// UTF-16 code units.
//
// A note is read by something that counts its own way, and a span named in
// bytes lands somewhere else in prose that is not ASCII.
func CountUTF16(text string, offset int) int {
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="setting offset to len(text) when offset equals len(text) is a no-op"
	if offset > len(text) {
		offset = len(text)
	}
	units := 0
	for _, r := range text[:offset] {
		units++
		if r > 0xffff {
			units++
		}
	}
	return units
}
