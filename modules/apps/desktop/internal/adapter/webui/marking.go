package webui

import (
	"slices"
	"strings"
	"unicode"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// marks is where each word typed stands in a passage, counted the way a client
// counts text: in UTF-16 code units.
//
// Case is folded and nothing else is. These marks are what says why a passage
// is here, and a mark on something the person did not type is a worse answer
// than no mark at all. A passage found for what it means rather than for what
// it says therefore carries none.
func marks(text, query string) []domain.Span {
	words := strings.Fields(query)
	if len(words) == 0 || text == "" {
		return nil
	}
	folded, units := folding(text)

	var at []domain.Span
	for _, word := range words {
		wanted, _ := folding(word)
		if len(wanted) == 0 {
			continue
		}
		for from := 0; from+len(wanted) <= len(folded); from++ {
			if slices.Equal(folded[from:from+len(wanted)], wanted) {
				at = append(at, domain.Span{From: units[from], To: units[from+len(wanted)]})
			}
		}
	}
	return at
}

// folding is the text one rune at a time with case dropped, and where each of
// those runes begins for something counting in UTF-16 code units.
//
// A rune at a time, because folding a whole string can change how many
// characters it holds and the offsets would no longer address the text they
// came from. The offsets run one longer than the text, so the end of the last
// rune is among them.
func folding(text string) ([]rune, []int) {
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

// How much of a passage is drawn, and how much of that stands before the run
// that matched, both counted in UTF-16 code units. The words a hit sits among
// are read from their own beginning and not from the middle of one.
const (
	glancing = 240
	leading  = 60
)

// around is the part of a passage worth drawing: the words about the first run
// that matched, or about the hit itself when no word matched at all.
//
// A passage is the whole of the window enclosing its hit, which for a note is
// the whole note. A hit by meaning stands on no word, so `from` is where the
// chunk that matched begins and is what the window opens near.
//
// The runs move with the text and the ones left outside are dropped, so what
// comes back addresses what comes back.
func around(text string, at []domain.Span, from int) (string, []domain.Span) {
	runes, units := counting(text)
	total := units[len(runes)]
	if total <= glancing {
		return text, at
	}

	// Where the window opens on: the first run that matched, and where the hit
	// itself stands when no word matched at all.
	point := from
	if len(at) > 0 {
		point = at[0].From
	}

	opens := 0
	if point > leading {
		opens = point - leading
	}
	to := opens + glancing
	if to > total {
		to, opens = total, total-glancing
	}
	first, last := begins(units, opens), ends(units, to)
	shift := units[first]

	var kept []domain.Span
	for _, span := range at {
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

// counting is the text one rune at a time, and where each of those runes begins
// for something counting in UTF-16 code units. The offsets run one longer than
// the text, so the end of the last rune is among them.
func counting(text string) ([]rune, []int) {
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

// begins is the rune the window opens on: the last one beginning at or before
// the offset given, so a cut never lands inside a character.
func begins(units []int, at int) int {
	for i := 1; i < len(units); i++ {
		if units[i] > at {
			return i - 1
		}
	}
	return len(units) - 1
}

// ends is the rune the window closes before: the first one beginning at or
// after the offset given.
func ends(units []int, at int) int {
	for i := range units {
		if units[i] >= at {
			return i
		}
	}
	return len(units) - 1
}
