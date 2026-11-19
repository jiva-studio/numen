package markdown

import (
	"unicode/utf8"
)

// Range is a run of prose, given by its start and end as byte offsets into the text
// it was found in. A client counts in other units (domain.UnitRange); CountUTF16
// converts, and no offset crosses that boundary unconverted.
type Range struct {
	Start int
	End   int
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
