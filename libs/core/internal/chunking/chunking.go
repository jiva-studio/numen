// Package chunking cuts a source's text into the chunks the index holds: a
// small chunk that carries a vector, and the large chunk enclosing it that a
// result shows.
//
// It is pure: no filesystem, no clock, no database. The same text and the same
// parts give the same offsets, so a chunk keeps an offset and not the text.
//
// Every offset is a byte offset into the text given, and a chunk's own text is
// text[Start : Start+Length]. A chunk is bounded in words, so a file that puts
// a whole book on one line is cut like any other.
//
// Chunks are cut inside one division, the text from one named part to the next,
// and never run across one. Sizes.Large set to Whole requests a single large
// chunk over the whole text, and a text that names no parts is one division.
//
// Sizes.Limit is characters, and the caller sets it under the input limit of the
// model that will embed a small chunk.
package chunking

import (
	"sort"
	"unicode"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Whole, as Sizes.Large, cuts one large chunk over the whole text: the chunk
// enclosing every small chunk is the source itself. This is how a note is cut.
// gomutants:disable-next-line INTEGER_INCREMENT reason="Whole is any negative size: Resolve maps every negative Large to Whole, so -2 behaves as -1"
const Whole = -1

// The sizes used where configuration names none.
const (
	DefaultLarge        = 200
	DefaultLargeOverlap = 40
	DefaultSmall        = 50
	DefaultSmallOverlap = 10
	DefaultLimit        = 1000
)

// The thresholds used where configuration names none.
const (
	DefaultAlphabetic = 0.65
	DefaultDirty      = 0.25
)

// CharactersPerToken is the floor a token is worth in characters.
//
// Latin runs about four; Devanagari and transliterated Sanskrit run under two, and
// Cyrillic between. Two is the floor, so a chunk cut under a model's limit by
// this is under it for every script — at the cost of a shorter chunk for Latin
// than the model could hold.
const CharactersPerToken = 2

// GetCharacterBound is what keeps a chunk inside a model's token limit. Zero
// tokens is a model that did not say, and takes the default bound.
func GetCharacterBound(tokens int) int {
	if tokens <= 0 {
		return DefaultLimit
	}
	return tokens * CharactersPerToken
}

// Sizes are how large a chunk is cut. A zero field takes its default.
type Sizes struct {
	// Large and Small are how many words a chunk of each size holds. Large is
	// Whole when one large chunk encloses the whole text.
	Large int
	Small int

	// LargeOverlap and SmallOverlap are how many words two consecutive chunks
	// of a size share. Each is at most one word short of its size, which
	// guarantees that tiling advances.
	LargeOverlap int
	SmallOverlap int

	// Limit is the most characters a small chunk may hold. A chunk over it is
	// cut further at a word, and a single word over it is not indexable.
	Limit int
}

// Legibility is what makes a chunk legible enough to index. A zero field takes
// its default, and a negative one disables the threshold.
type Legibility struct {
	// Alphabetic is the least fraction of a chunk's characters that must be
	// letters.
	Alphabetic float64

	// Dirty is the most fraction of a chunk's words that may carry a non-letter
	// inside.
	Dirty float64
}

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

// ExtractText returns the chunk's own text from the text it was cut from.
func (c Chunk) ExtractText(text string) string { return text[c.Start : c.Start+c.Length] }

// computeMiddle returns the byte the chunk is centred on; it decides which large
// chunk a small one belongs to.
func (c Chunk) computeMiddle() int { return c.Start + c.Length/2 }

// Chunks cuts the text and returns every chunk as the index stores it. The
// build function creates one chunk from a cut and receives whether it is a
// large chunk or a small one inside it.
func Chunks(
	text string, parts []domain.PartStart, sizes Sizes, reads Legibility,
	build func(c Chunk, isLarge bool) domain.Chunk,
) []domain.Chunk {
	var out []domain.Chunk
	for _, large := range Cut(text, parts, sizes, reads) {
		c := build(large, true)
		for _, small := range large.Small {
			c.Small = append(c.Small, build(small, false))
		}
		out = append(out, c)
	}
	return out
}

// Cut returns the large chunks of the text, each carrying the small chunks
// inside it.
func Cut(text string, parts []domain.PartStart, sizes Sizes, reads Legibility) []Chunk {
	s, l := sizes.Resolve(), reads.resolve()
	divisions := divideText(text, parts)

	if s.Large == Whole {
		return createWholeTextChunks(text, divisions, s, l)
	}
	var out []Chunk
	for _, d := range divisions {
		out = append(out, splitDivision(text, d, s, l)...)
	}
	return out
}

// A division is the text one named part covers: from that part to the next.
type division struct {
	from, to int
	location string
}

// divideText divides the text at the parts it names. The parts are copied
// before they are ordered, so Cut leaves its arguments unmodified.
// Where two parts share an offset, the last of them names the text after it.
func divideText(text string, parts []domain.PartStart) []division {
	named := make([]domain.PartStart, 0, len(parts))
	for _, p := range parts {
		if p.Offset >= 0 && p.Offset < len(text) {
			named = append(named, p)
		}
	}
	sort.SliceStable(named, func(i, j int) bool { return named[i].Offset < named[j].Offset })

	var divisions []division
	pos, location := 0, ""
	for _, p := range named {
		if p.Offset > pos {
			divisions = append(divisions, division{from: pos, to: p.Offset, location: location})
		}
		pos, location = p.Offset, p.Title
	}
	return append(divisions, division{from: pos, to: len(text), location: location})
}

// splitDivision tiles one division twice and puts each small chunk under the large
// one its middle falls in.
func splitDivision(text string, d division, s Sizes, l Legibility) []Chunk {
	words := splitWords(text, d.from, d.to)
	if len(words) == 0 {
		return nil
	}
	var large []Chunk
	for _, item := range getOverlappingRanges(len(words), s.Large, s.LargeOverlap) {
		c := getRangeChunk(words, item, d.location)
		if !isLegible(c.ExtractText(text), l) {
			continue
		}
		large = append(large, c)
	}
	return enclose(large, createSmallChunks(text, words, d.location, s, l))
}

// createWholeTextChunks makes the text itself the large chunk and cuts the small chunks on the
// structure inside it.
func createWholeTextChunks(text string, divisions []division, s Sizes, l Legibility) []Chunk {
	words := splitWords(text, 0, len(text))
	if len(words) == 0 {
		return nil
	}
	large := getRangeChunk(words, [2]int{0, len(words)}, "")
	if !isLegible(large.ExtractText(text), l) {
		return nil
	}
	var small []Chunk
	start := 0
	for _, d := range divisions {
		for start < len(words) && words[start].start < d.from {
			start++
		}
		end := start
		for end < len(words) && words[end].end <= d.to {
			end++
		}
		if start < end {
			small = append(small, createSmallChunks(text, words[start:end], d.location, s, l)...)
		}
	}
	return enclose([]Chunk{large}, small)
}

// createSmallChunks tiles the words of one division into the chunks that carry a
// vector. A chunk over the character limit is cut further at a word, and one
// that is still over it after that is a single word and is dropped.
func createSmallChunks(text string, words []word, location string, s Sizes, l Legibility) []Chunk {
	var out []Chunk
	for _, item := range getOverlappingRanges(len(words), s.Small, s.SmallOverlap) {
		for _, piece := range splitAtLimit(text, words, item, s.Limit) {
			c := getRangeChunk(words, piece, location)
			body := c.ExtractText(text)
			if utf8.RuneCountInString(body) > s.Limit || !isLegible(body, l) {
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
// part of it is what a result displays, and the words after the part it holds
// are the rest of the sentence the hit is in.
//
// Large chunks overlap, so where none holds the whole of it the middle decides,
// and that is the first large chunk reaching past it. Both lists ascend and the
// look forward stops at the middle, so one pass over each is enough.
func enclose(large, small []Chunk) []Chunk {
	pos := 0
	for _, c := range small {
		for pos < len(large) && large[pos].Start+large[pos].Length <= c.computeMiddle() {
			pos++
		}
		if pos == len(large) {
			break
		}
		if large[pos].Start > c.computeMiddle() {
			continue
		}
		underVal := pos
		for i := pos; i < len(large) && large[i].Start <= c.computeMiddle(); i++ {
			if large[i].Start <= c.Start && c.Start+c.Length <= large[i].Start+large[i].Length {
				underVal = i
				break
			}
		}
		large[underVal].Small = append(large[underVal].Small, c)
	}
	return large
}

// A word is the byte extent of one run of characters that is not space.
type word struct{ start, end int }

// splitWords splits text[from:to] into its words.
func splitWords(text string, from, to int) []word {
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

// getOverlappingRanges covers n words with ranges of size words sharing overlap of them. The
// last range ends at n, so a division shorter than one chunk is one chunk.
func getOverlappingRanges(n, size, overlap int) [][2]int {
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

// splitAtLimit breaks a range of words into pieces of at most limit characters,
// counting the space between words. A word of its own is a piece however long it
// is.
func splitAtLimit(text string, words []word, pos [2]int, limit int) [][2]int {
	from, to := pos[0], pos[1]
	if utf8.RuneCountInString(text[words[from].start:words[to-1].end]) <= limit {
		return [][2]int{pos}
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

// getRangeChunk is the chunk covering a range of words.
func getRangeChunk(words []word, pos [2]int, location string) Chunk {
	start, end := words[pos[0]].start, words[pos[1]-1].end
	return Chunk{Start: start, Length: end - start, Location: location}
}

// Resolve fills in what configuration left unset and keeps every size usable.
// It is the sizes Cut works to, and what a caller names them by.
func (s Sizes) Resolve() Sizes {
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
	if s.Large < 0 {
		s.Large = Whole
	}
	s.LargeOverlap = clampOverlap(s.LargeOverlap, s.Large)
	s.SmallOverlap = clampOverlap(s.SmallOverlap, s.Small)
	return s
}

// resolve fills in what configuration left unset.
func (l Legibility) resolve() Legibility {
	if l.Alphabetic == 0 {
		l.Alphabetic = DefaultAlphabetic
	}
	if l.Dirty == 0 {
		l.Dirty = DefaultDirty
	}
	return l
}

// clampOverlap keeps an overlap between none and one word short of the size it belongs
// to, which guarantees that tiling advances by at least one word.
func clampOverlap(overlap, size int) int {
	if overlap < 0 {
		return 0
	}
	if size > 0 && overlap >= size {
		return size - 1
	}
	return overlap
}
