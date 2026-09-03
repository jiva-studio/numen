// Package chunking cuts a source's text into the chunks the index holds: a
// small chunk that carries a vector, and the large chunk enclosing it that a
// result shows.
//
// It is pure: no filesystem, no clock, no database. The same text and the same
// parts give the same offsets, which is what lets a chunk keep an offset and
// not the text.
//
// Every offset is a byte offset into the text given, and a chunk's own text is
// text[Start : Start+Length]. A chunk is bounded in words, so a file that puts
// a whole book on one line is cut like any other.
//
// Chunks are cut inside one division — the text from one named part to the next
// — and never run across one. A text that names no parts is one division, which
// is the ordinary case for half the books read.
//
// Sizes.Limit is characters, and the caller sets it under the input limit of the
// model that will embed a small chunk. What a word costs in tokens differs by
// script and is recorded in docs/performance.md.
package chunking

import (
	"sort"
	"unicode"
	"unicode/utf8"
)

// Whole, as Sizes.Large, cuts one large chunk over the whole text: the chunk
// enclosing every small chunk is the source itself. This is how a note is cut.
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
// Cyrillic between. Two is the floor, so a chunk cut under a model's limit by
// this is under it for every script — at the cost of a shorter chunk for Latin
// than the model could hold.
const CharactersPerToken = 2

// Under is the character bound that keeps a chunk inside a model's token limit.
// Zero tokens is a model that did not say, and takes the default bound.
func Under(tokens int) int {
	if tokens <= 0 {
		return DefaultLimit
	}
	return tokens * CharactersPerToken
}

// Sizes are how large a chunk is cut and what makes a chunk legible enough to
// index. A zero field takes its default.
type Sizes struct {
	// Large and Small are how many words a chunk of each size holds. Large is
	// Whole when one large chunk encloses the whole text.
	Large int
	Small int

	// LargeOverlap and SmallOverlap are how many words two consecutive chunks
	// of a size share. Each is at most one word short of its size, so that
	// tiling advances.
	LargeOverlap int
	SmallOverlap int

	// Limit is the most characters a small chunk may hold. A chunk over it is
	// cut further at a word, and a single word over it is not indexable.
	Limit int

	// Alphabetic is the least fraction of a chunk's characters that must be
	// letters, and Dirty the most fraction of its words that may carry a
	// non-letter inside. A negative value asks for no threshold.
	Alphabetic float64
	Dirty      float64
}

// A PartStart is somewhere in the text that carries a name. Parts bound the
// divisions chunks are cut inside, and need not arrive in order.
type PartStart struct {
	Title  string
	Offset int
}

// A Chunk is one cut of the text. Location is where the chunk sits in the terms
// the source's own numbering uses, and is empty where the text named none.
//
// Small are the chunks inside this one. A Chunk with none of its own is a large
// chunk all the same: what makes it large is that nothing encloses it.
type Chunk struct {
	Start    int
	Length   int
	Location string
	Small    []Chunk
}

// Slice is the chunk's own text.
func (c Chunk) Slice(text string) string { return text[c.Start : c.Start+c.Length] }

// middle is the byte the chunk is centred on, and is what decides which large
// chunk a small one belongs to.
func (c Chunk) middle() int { return c.Start + c.Length/2 }

// Cut returns the large chunks of the text, each carrying the small chunks
// inside it.
func Cut(text string, parts []PartStart, sizes Sizes) []Chunk {
	s := sizes.resolve()
	divisions := divisionsOf(text, parts)

	if s.Large == Whole {
		return whole(text, divisions, s)
	}
	var out []Chunk
	for _, d := range divisions {
		out = append(out, cutDivision(text, d, s)...)
	}
	return out
}

// A division is the text one named part covers: from that part to the next.
type division struct {
	from, to int
	location string
}

// divisionsOf divides the text at the parts it names. The parts are copied
// before they are ordered, so that Cut leaves its arguments as it found them.
// Where two parts share an offset, the last of them names the text after it.
func divisionsOf(text string, parts []PartStart) []division {
	named := make([]PartStart, 0, len(parts))
	for _, p := range parts {
		if p.Offset >= 0 && p.Offset < len(text) {
			named = append(named, p)
		}
	}
	sort.SliceStable(named, func(i, j int) bool { return named[i].Offset < named[j].Offset })

	var divisions []division
	at, location := 0, ""
	for _, p := range named {
		if p.Offset > at {
			divisions = append(divisions, division{from: at, to: p.Offset, location: location})
		}
		at, location = p.Offset, p.Title
	}
	return append(divisions, division{from: at, to: len(text), location: location})
}

// cutDivision tiles one division twice and puts each small chunk under the large
// one its middle falls in.
func cutDivision(text string, d division, s Sizes) []Chunk {
	words := wordsIn(text, d.from, d.to)
	if len(words) == 0 {
		return nil
	}
	var large []Chunk
	for _, at := range tile(len(words), s.Large, s.LargeOverlap) {
		c := extent(words, at, d.location)
		if !legible(c.Slice(text), s) {
			continue
		}
		large = append(large, c)
	}
	return enclose(large, smallChunks(text, words, d.location, s))
}

// whole makes the text itself the large chunk and cuts the small chunks on the
// structure inside it.
func whole(text string, divisions []division, s Sizes) []Chunk {
	words := wordsIn(text, 0, len(text))
	if len(words) == 0 {
		return nil
	}
	large := extent(words, [2]int{0, len(words)}, "")
	if !legible(large.Slice(text), s) {
		return nil
	}
	var small []Chunk
	for _, d := range divisions {
		small = append(small, smallChunks(text, wordsIn(text, d.from, d.to), d.location, s)...)
	}
	return enclose([]Chunk{large}, small)
}

// smallChunks tiles the words of one division into the chunks that carry a
// vector. A chunk over the character limit is cut further at a word, and one
// that is still over it after that is a single word and is dropped.
func smallChunks(text string, words []word, location string, s Sizes) []Chunk {
	var out []Chunk
	for _, at := range tile(len(words), s.Small, s.SmallOverlap) {
		for _, piece := range limited(text, words, at, s.Limit) {
			c := extent(words, piece, location)
			body := c.Slice(text)
			if utf8.RuneCountInString(body) > s.Limit || !legible(body, s) {
				continue
			}
			out = append(out, c)
		}
	}
	return out
}

// enclose puts each small chunk under a large chunk holding it.
//
// The one preferred holds the small chunk whole. A large chunk holding only
// part of it is what a person is shown, and the words after the part it holds
// are the rest of the sentence the hit is in.
//
// Large chunks overlap, so where none holds the whole of it the middle decides,
// and that is the first large chunk reaching past it. Both lists ascend and the
// look forward stops at the middle, so one pass over each is enough.
func enclose(large, small []Chunk) []Chunk {
	at := 0
	for _, c := range small {
		for at < len(large) && large[at].Start+large[at].Length <= c.middle() {
			at++
		}
		if at == len(large) {
			break
		}
		if large[at].Start > c.middle() {
			continue
		}
		under := at
		for i := at; i < len(large) && large[i].Start <= c.middle(); i++ {
			if large[i].Start <= c.Start && c.Start+c.Length <= large[i].Start+large[i].Length {
				under = i
				break
			}
		}
		large[under].Small = append(large[under].Small, c)
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
// last range ends at n, so a division shorter than one chunk is one chunk.
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

// extent is the chunk covering a range of words.
func extent(words []word, at [2]int, location string) Chunk {
	start, end := words[at[0]].start, words[at[1]-1].end
	return Chunk{Start: start, Length: end - start, Location: location}
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
