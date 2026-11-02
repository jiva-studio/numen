package search

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// counts text: in UTF-16 code units. The spans come in order and do not
// overlap.
//
// Case is folded and nothing else is. A span begins where a word begins, and
// ends where one ends; the last word typed may still be growing, so it matches
// a word by its opening, which is the rule the index matched it by.
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

// isWordRune reports whether r can be part of a word, so the runes adjacent to
// a match distinguish a whole word from letters inside one. Chinese, Japanese
// and Korean characters are excluded: those scripts have no word-internal
// letters to distinguish from a boundary, so every character is a word.
func isWordRune(r rune) bool { return (unicode.IsLetter(r) || unicode.IsDigit(r)) && !isCJKRune(r) }

// isCJKRune is a character of a Chinese, Japanese or Korean script.
func isCJKRune(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r)
}

// isWordStart and isWordEnd say whether a span beginning or ending here is a whole
// word's beginning or end. The ends of the text are both.
func isWordStart(runes []rune, pos int) bool { return pos == 0 || !isWordRune(runes[pos-1]) }

func isWordEnd(runes []rune, pos int) bool { return pos == len(runes) || !isWordRune(runes[pos]) }

// mergeRanges is the spans in the order they stand, with ones that touch or overlap
// made into one. Two words typed can name the same characters.
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

// getFoldedRunes is the text one rune at a time with case dropped, and where each of
// those runes begins for something counting in UTF-16 code units.
//
// A rune at a time: folding a whole string can change how many characters it
// holds. The offsets run one longer than the text, so the end of the last rune
// is among them.
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

// glancing is the maximum passage length and leading is how much of it may
// precede the matched span, both in UTF-16 code units. Matched words are cut at
// word starts, not mid-word.
const (
	glancing = 240
	leading  = 60
)

// getTextAround is the part of a passage worth drawing: the words about the
// first span that matched, or about the hit itself when no word matched at all.
//
// A passage is the whole of the window enclosing its hit, which for a note is
// the whole note. A hit by meaning relies on no word, so `from` is where the
// chunk that matched begins and is what the window opens near.
//
// The spans move with the text and the ones left outside are dropped, so what
// comes back addresses what comes back.
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
	// A window reaching past the end of the passage closes there and holds less
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

// getRunes is the text one rune at a time, and where each of those runes begins
// for something counting in UTF-16 code units. The offsets run one longer than
// the text, so the end of the last rune is among them.
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

// getOpeningRune is the rune the window opens on: the last one beginning at or
// before the offset given, so a cut never lands inside a character.
func getOpeningRune(units []int, pos int) int {
	for i := 1; i < len(units); i++ {
		if units[i] > pos {
			return i - 1
		}
	}
	return len(units) - 1
}

// getClosingRune is the rune the window closes before: the first one beginning
// at or after the offset given.
func getClosingRune(units []int, pos int) int {
	for i := range units {
		if units[i] >= pos {
			return i
		}
	}
	return len(units) - 1
}

// reach is how far on either side of a hit a passage is read, in bytes. The
// window opens slightly before the first matched word and extends from there, so
// the range read must contain the widest window that can open near the hit and
// the words that can open it.
const reach = 8 * glancing

// nearby is the part of a passage the window is cut from, and where the hit
//
// A passage is the whole of the window enclosing its hit, which for a section
// of a book is the whole section. What is read is what a window can be cut
// from.
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
