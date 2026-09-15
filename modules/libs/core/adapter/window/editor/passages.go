package editor

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// spans is where each word typed stands in a passage, counted the way a client
// counts text: in UTF-16 code units. The spans come in order and do not
// overlap.
//
// Case is folded and nothing else is. A span begins where a word begins, and
// ends where one ends; the last word typed may still be growing, so it matches
// a word by its opening, which is the rule the index matched it by.
func spans(text, query string) []domain.Span {
	words := strings.Fields(query)
	if len(words) == 0 || text == "" {
		return nil
	}
	folded, units := getFoldedRunes(text)

	var at []domain.Span
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
			if !opens(folded, from) || (whole && !closes(folded, to)) {
				continue
			}
			at = append(at, domain.Span{From: units[from], To: units[to]})
		}
	}
	return mergeSpans(at)
}

// wordly is what a word is made of, so that what stands either side of one is
// what tells a word from a run of letters inside one.
func wordly(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// opens and closes say whether a span beginning or ending here is a whole
// word's beginning or end. The ends of the text are both.
func opens(runes []rune, at int) bool { return at == 0 || !wordly(runes[at-1]) }

func closes(runes []rune, at int) bool { return at == len(runes) || !wordly(runes[at]) }

// mergeSpans is the spans in the order they stand, with ones that touch or overlap
// made into one. Two words typed can name the same characters.
func mergeSpans(at []domain.Span) []domain.Span {
	if len(at) < 2 {
		return at
	}
	slices.SortFunc(at, func(a, b domain.Span) int { return a.From - b.From })

	out := at[:1]
	for _, span := range at[1:] {
		last := &out[len(out)-1]
		if span.From <= last.To {
			last.To = max(last.To, span.To)
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

	at := 0
	for _, r := range text {
		runes = append(runes, unicode.ToLower(r))
		units = append(units, at)
		at++
		if r > 0xffff {
			at++
		}
	}
	return runes, append(units, at)
}

// How much of a passage is drawn at most, and how much of that stands before
// the span that matched, both counted in UTF-16 code units. The words a hit sits
// among are read from their own beginning and not from the middle of one.
const (
	glancing = 240
	leading  = 60
)

// getTextAround is the part of a passage worth drawing: the words about the
// first span that matched, or about the hit itself when no word matched at all.
//
// A passage is the whole of the window enclosing its hit, which for a note is
// the whole note. A hit by meaning stands on no word, so `from` is where the
// chunk that matched begins and is what the window opens near.
//
// The spans move with the text and the ones left outside are dropped, so what
// comes back addresses what comes back.
func getTextAround(text string, spans []domain.Span, from int) (string, []domain.Span) {
	runes, units := getRunes(text)
	total := units[len(runes)]
	if total <= glancing {
		return text, spans
	}

	// Where the window opens on: the first span that matched, and where the hit
	// itself stands when no word matched at all.
	point := from
	if len(spans) > 0 {
		point = spans[0].From
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

	var kept []domain.Span
	for _, span := range spans {
		clipped := domain.Span{From: max(span.From, units[first]), To: min(span.To, units[last])}
		if clipped.From >= clipped.To {
			continue
		}
		kept = append(kept, domain.Span{From: clipped.From - shift, To: clipped.To - shift})
	}

	cut := string(runes[first:last])
	if first > 0 {
		// The mark for what was left off stands before the text, so everything
		// in it begins one unit further along.
		cut = "…" + cut
		for i := range kept {
			kept[i] = domain.Span{From: kept[i].From + 1, To: kept[i].To + 1}
		}
	}
	if last < len(runes) {
		cut += "…"
	}
	return cut, kept
}

// getRunes is the text one rune at a time, and where each of those runes begins
// for something counting in UTF-16 code units. The offsets run one longer than
// the text, so the end of the last rune is among them.
func getRunes(text string) ([]rune, []int) {
	runes := make([]rune, 0, len(text))
	units := make([]int, 0, len(text)+1)

	at := 0
	for _, r := range text {
		runes = append(runes, r)
		units = append(units, at)
		at++
		if r > 0xffff {
			at++
		}
	}
	return runes, append(units, at)
}

// getOpeningRune is the rune the window opens on: the last one beginning at or
// before the offset given, so a cut never lands inside a character.
func getOpeningRune(units []int, at int) int {
	for i := 1; i < len(units); i++ {
		if units[i] > at {
			return i - 1
		}
	}
	return len(units) - 1
}

// getClosingRune is the rune the window closes before: the first one beginning
// at or after the offset given.
func getClosingRune(units []int, at int) int {
	for i := range units {
		if units[i] >= at {
			return i
		}
	}
	return len(units) - 1
}

// How far either side of a hit a passage is read at all, in bytes. The window
// drawn opens a little before the first word that matched and runs on from
// there, so what is read has to hold the widest window that can open near the
// hit and the words that can open it.
const reach = 8 * glancing

// nearby is the part of a passage the window is cut from, and where the hit
// stands inside it.
//
// A passage is the whole of the window enclosing its hit, which for a section
// of a book is the whole section. What is read is what a window can be cut
// from.
func nearby(text string, hit int) (string, int) {
	from, to := hit-reach, hit+reach
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
	return text[from:to], hit - from
}
