// Package window cuts a source's text into the chunks the index holds: a small
// window that carries a vector, and the large window enclosing it that a result
// shows.
//
// It is pure: no filesystem, no clock, no database. The same text and the same
// places give the same offsets, which is what lets a chunk keep an offset and
// not the text.
//
// Every offset is a byte offset into the text given, and a window's own text is
// text[Start : Start+Length]. A window is bounded in words, so a file that puts
// a whole book on one line is cut like any other.
//
// Windows are cut inside a span — the text from one named place to the next —
// and never run across one. A text that names no places is one span, which is
// the ordinary case for half the books read.
//
// Sizes.Limit is characters, and the caller sets it under the input limit of the
// model that will embed a small window. What a word costs in tokens differs by
// script and is recorded in docs/performance.md.
package window

import (
	"sort"
	"unicode"
	"unicode/utf8"
)

// Whole, as Sizes.Large, cuts one large window over the whole text: the window
// enclosing every small window is the source itself. This is how a note is cut.
const Whole = -1

// The sizes and thresholds used where configuration names none.
const (
	DefaultLarge        = 200
	DefaultLargeOverlap = 40
	DefaultSmall        = 50
	DefaultSmallOverlap = 10
	DefaultLimit        = 1000
	DefaultAlphabetic   = 0.65
	DefaultDirty        = 0.25
)

// CharactersPerToken is the floor a token is worth in characters.
//
// Latin runs about four; Devanagari and transliterated Sanskrit run under two, and
// Cyrillic between. Two is the floor, so a window cut under a model's limit by
// this is under it for every script — at the cost of a shorter window for Latin
// than the model could hold.
const CharactersPerToken = 2

// Under is the character bound that keeps a window inside a model's token limit.
// Zero tokens is a model that did not say, and takes the default bound.
func Under(tokens int) int {
	if tokens <= 0 {
		return DefaultLimit
	}
	return tokens * CharactersPerToken
}

// Sizes are how large a chunk is cut and what makes a window legible enough to
// index. A zero field takes its default.
type Sizes struct {
	// Large and Small are how many words a window of each size holds. Large is
	// Whole when one large window encloses the whole text.
	Large int
	Small int

	// LargeOverlap and SmallOverlap are how many words two consecutive windows
	// of a size share. Each is at most one word short of its size, so that
	// tiling advances.
	LargeOverlap int
	SmallOverlap int

	// Limit is the most characters a small window may hold. A window over it is
	// cut further at a word, and a single word over it is not indexable.
	Limit int

	// Alphabetic is the least fraction of a window's characters that must be
	// letters, and Dirty the most fraction of its words that may carry a
	// non-letter inside. A negative value asks for no threshold.
	Alphabetic float64
	Dirty      float64
}

// A Place is somewhere in the text that carries a name. Places bound the spans
// windows are cut inside, and need not arrive in order.
type Place struct {
	Title  string
	Offset int
}

// A Window is one cut of the text. Location is what the source's own numbering
// calls the place the window sits in, and is empty where the text named none.
//
// Small are the windows inside this one. A Window with none of its own is a
// large window all the same: what makes it large is that nothing encloses it.
type Window struct {
	Start    int
	Length   int
	Location string
	Small    []Window
}

// Slice is the window's own text.
func (w Window) Slice(text string) string { return text[w.Start : w.Start+w.Length] }

// middle is the byte the window is centred on, and is what decides which large
// window a small one belongs to.
func (w Window) middle() int { return w.Start + w.Length/2 }

// Cut returns the large windows of the text, each carrying the small windows
// inside it.
func Cut(text string, places []Place, sizes Sizes) []Window {
	s := sizes.resolve()
	spans := spansOf(text, places)

	if s.Large == Whole {
		return whole(text, spans, s)
	}
	var out []Window
	for _, sp := range spans {
		out = append(out, cutSpan(text, sp, s)...)
	}
	return out
}

// A span is the text one named place covers: from that place to the next.
type span struct {
	from, to int
	location string
}

// spansOf divides the text at the places it names. The places are copied before
// they are ordered, so that Cut leaves its arguments as it found them. Where two
// places share an offset, the last of them names the text after it.
func spansOf(text string, places []Place) []span {
	named := make([]Place, 0, len(places))
	for _, p := range places {
		if p.Offset >= 0 && p.Offset < len(text) {
			named = append(named, p)
		}
	}
	sort.SliceStable(named, func(i, j int) bool { return named[i].Offset < named[j].Offset })

	var spans []span
	at, location := 0, ""
	for _, p := range named {
		if p.Offset > at {
			spans = append(spans, span{from: at, to: p.Offset, location: location})
		}
		at, location = p.Offset, p.Title
	}
	return append(spans, span{from: at, to: len(text), location: location})
}

// cutSpan tiles one span twice and puts each small window under the large one
// its middle falls in.
func cutSpan(text string, sp span, s Sizes) []Window {
	words := wordsIn(text, sp.from, sp.to)
	if len(words) == 0 {
		return nil
	}
	var large []Window
	for _, at := range tile(len(words), s.Large, s.LargeOverlap) {
		w := extent(words, at, sp.location)
		if !legible(w.Slice(text), s) {
			continue
		}
		large = append(large, w)
	}
	return enclose(large, smallWindows(text, words, sp.location, s))
}

// whole makes the text itself the large window and cuts the small windows on the
// structure inside it.
func whole(text string, spans []span, s Sizes) []Window {
	words := wordsIn(text, 0, len(text))
	if len(words) == 0 {
		return nil
	}
	large := extent(words, [2]int{0, len(words)}, "")
	if !legible(large.Slice(text), s) {
		return nil
	}
	var small []Window
	for _, sp := range spans {
		small = append(small, smallWindows(text, wordsIn(text, sp.from, sp.to), sp.location, s)...)
	}
	return enclose([]Window{large}, small)
}

// smallWindows tiles the words of one span into the windows that carry a vector.
// A window over the character limit is cut further at a word, and one that is
// still over it after that is a single word and is dropped.
func smallWindows(text string, words []word, location string, s Sizes) []Window {
	var out []Window
	for _, at := range tile(len(words), s.Small, s.SmallOverlap) {
		for _, piece := range limited(text, words, at, s.Limit) {
			w := extent(words, piece, location)
			body := w.Slice(text)
			if utf8.RuneCountInString(body) > s.Limit || !legible(body, s) {
				continue
			}
			out = append(out, w)
		}
	}
	return out
}

// enclose puts each small window under the first large window holding its
// middle. Both lists ascend, so one pass over each is enough. A small window that
// no large window holds is produced only where a large window holds its middle.
func enclose(large, small []Window) []Window {
	at := 0
	for _, w := range small {
		for at < len(large) && large[at].Start+large[at].Length <= w.middle() {
			at++
		}
		if at == len(large) {
			break
		}
		if large[at].Start <= w.middle() {
			large[at].Small = append(large[at].Small, w)
		}
	}
	return large
}

// A word is the byte extent of one run of characters that is not space.
type word struct{ start, end int }

// wordsIn finds the words of text[from:to].
func wordsIn(text string, from, to int) []word {
	var words []word
	start := -1
	for i, r := range text[from:to] {
		if unicode.IsSpace(r) {
			if start >= 0 {
				words = append(words, word{start: from + start, end: from + i})
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		words = append(words, word{start: from + start, end: to})
	}
	return words
}

// tile covers n words with ranges of size words sharing overlap of them. The
// last range ends at n, so a span shorter than one window is one window.
func tile(n, size, overlap int) [][2]int {
	step := size - overlap
	var out [][2]int
	for i := 0; i < n; i += step {
		end := min(i+size, n)
		out = append(out, [2]int{i, end})
		if end == n {
			break
		}
	}
	return out
}

// limited breaks a range of words into pieces of at most limit characters,
// counting the space between words. A word of its own is a piece however long it
// is.
func limited(text string, words []word, at [2]int, limit int) [][2]int {
	from, to := at[0], at[1]
	if utf8.RuneCountInString(text[words[from].start:words[to-1].end]) <= limit {
		return [][2]int{at}
	}
	var out [][2]int
	start, chars := from, 0
	for i := from; i < to; i++ {
		n := utf8.RuneCountInString(text[words[i].start:words[i].end]) + 1
		if i > start && chars+n > limit {
			out = append(out, [2]int{start, i})
			start, chars = i, 0
		}
		chars += n
	}
	return append(out, [2]int{start, to})
}

// extent is the window covering a range of words.
func extent(words []word, at [2]int, location string) Window {
	start, end := words[at[0]].start, words[at[1]-1].end
	return Window{Start: start, Length: end - start, Location: location}
}

// resolve fills in what configuration left unset and keeps every size usable.
func (s Sizes) resolve() Sizes {
	if s.Large == 0 {
		s.Large = DefaultLarge
	}
	if s.Small <= 0 {
		s.Small = DefaultSmall
	}
	if s.LargeOverlap == 0 {
		s.LargeOverlap = DefaultLargeOverlap
	}
	if s.SmallOverlap == 0 {
		s.SmallOverlap = DefaultSmallOverlap
	}
	if s.Limit <= 0 {
		s.Limit = DefaultLimit
	}
	if s.Alphabetic == 0 {
		s.Alphabetic = DefaultAlphabetic
	}
	if s.Dirty == 0 {
		s.Dirty = DefaultDirty
	}
	if s.Large < 0 {
		s.Large = Whole
	}
	s.LargeOverlap = bound(s.LargeOverlap, s.Large)
	s.SmallOverlap = bound(s.SmallOverlap, s.Small)
	return s
}

// bound keeps an overlap between none and one word short of the size it belongs
// to, so that tiling advances by at least one word.
func bound(overlap, size int) int {
	if overlap < 0 {
		return 0
	}
	if size > 0 && overlap >= size {
		return size - 1
	}
	return overlap
}
