package search

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/markdown"
)

// CutSnippet returns the window around the hit of a search result and the matched query spans inside it.
func CutSnippet(text string, matchOffset int, query string) (string, []domain.UnitRange) {
	read, matched := nearby(text, matchOffset)
	return getTextAround(read, spans(read, query), markdown.CountUTF16(read, matched))
}

func spans(text, query string) []domain.UnitRange {
	words := strings.Fields(query)
	if len(words) == 0 || text == "" {
		return nil
	}
	folded, units := getFoldedRunes(text)

	var offset []domain.UnitRange
	for i, word := range words {
		wanted, _ := getFoldedRunes(word)
		if len(wanted) == 0 {
			continue
		}
		whole := i < len(words)-1
		for from := 0; from+len(wanted) <= len(folded); from++ {
			to := from + len(wanted)
			if !slices.Equal(folded[from:to], wanted) {
				continue
			}
			if !isWordStart(folded, from) || (whole && !isWordEnd(folded, to)) {
				continue
			}
			offset = append(offset, domain.UnitRange{Start: units[from], End: units[to]})
		}
	}
	return mergeRanges(offset)
}

func isWordRune(r rune) bool { return (unicode.IsLetter(r) || unicode.IsDigit(r)) && !isCJKRune(r) }

func isCJKRune(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r)
}

func isWordStart(runes []rune, pos int) bool { return pos == 0 || !isWordRune(runes[pos-1]) }

func isWordEnd(runes []rune, pos int) bool { return pos == len(runes) || !isWordRune(runes[pos]) }

func mergeRanges(pos []domain.UnitRange) []domain.UnitRange {
	if len(pos) < 2 {
		return pos
	}
	slices.SortFunc(pos, func(a, b domain.UnitRange) int { return a.Start - b.Start })

	out := pos[:1]
	for _, span := range pos[1:] {
		last := &out[len(out)-1]
		if span.Start <= last.End {
			last.End = max(last.End, span.End)
			continue
		}
		out = append(out, span)
	}
	return out
}

func getFoldedRunes(text string) ([]rune, []int) {
	runes := make([]rune, 0, len(text))
	units := make([]int, 0, len(text)+1)

	pos := 0
	for _, r := range text {
		runes = append(runes, unicode.ToLower(r))
		units = append(units, pos)
		pos++
		if r > 0xffff {
			pos++
		}
	}
	return runes, append(units, pos)
}

// glancing is the maximum search result length and leading is how much of it may
// precede the matched span, both in UTF-16 code units. Matched words are cut at
// word starts, not mid-word.
const (
	glancing = 240
	leading  = 60
)

func getTextAround(text string, spans []domain.UnitRange, from int) (string, []domain.UnitRange) {
	runes, units := getRunes(text)
	total := units[len(runes)]
	if total <= glancing {
		return text, spans
	}

	// The window starts at the first matched span, or at the hit itself when no
	// word matched.
	point := from
	if len(spans) > 0 {
		point = spans[0].Start
	}

	opens := 0
	if point > leading {
		opens = point - leading
	}
	// A window reaching past the end of the search result closes there and holds less
	// than a glance.
	to := min(opens+glancing, total)
	first, last := getOpeningRune(units, opens), getClosingRune(units, to)
	shift := units[first]

	var clippedSpans []domain.UnitRange
	for _, span := range spans {
		clipped := domain.UnitRange{Start: max(span.Start, units[first]), End: min(span.End, units[last])}
		if clipped.Start >= clipped.End {
			continue
		}
		clippedSpans = append(clippedSpans, domain.UnitRange{Start: clipped.Start - shift, End: clipped.End - shift})
	}

	snippet := string(runes[first:last])
	if first > 0 {
		// The ellipsis prepended to the text shifts every span in it one unit right.
		snippet = "…" + snippet
		for i := range clippedSpans {
			clippedSpans[i] = domain.UnitRange{Start: clippedSpans[i].Start + 1, End: clippedSpans[i].End + 1}
		}
	}
	if last < len(runes) {
		snippet += "…"
	}
	return snippet, clippedSpans
}

func getRunes(text string) ([]rune, []int) {
	runes := make([]rune, 0, len(text))
	units := make([]int, 0, len(text)+1)

	pos := 0
	for _, r := range text {
		runes = append(runes, r)
		units = append(units, pos)
		pos++
		if r > 0xffff {
			pos++
		}
	}
	return runes, append(units, pos)
}

func getOpeningRune(units []int, pos int) int {
	for i := 1; i < len(units); i++ {
		if units[i] > pos {
			return i - 1
		}
	}
	return len(units) - 1
}

func getClosingRune(units []int, pos int) int {
	for i := range units {
		if units[i] >= pos {
			return i
		}
	}
	return len(units) - 1
}

// reach is how far on either side of a hit a search result is read, in bytes. The
// window opens slightly before the first matched word and extends from there, so
// the range read must contain the widest window that can open near the hit and
// the words that can open it.
const reach = 8 * glancing

func nearby(text string, index int) (string, int) {
	from, to := index-reach, index+reach
	if from < 0 {
		from = 0
	}
	if to > len(text) {
		to = len(text)
	}
	for from > 0 && !utf8.RuneStart(text[from]) {
		from--
	}
	for to < len(text) && !utf8.RuneStart(text[to]) {
		to++
	}
	return text[from:to], index - from
}
