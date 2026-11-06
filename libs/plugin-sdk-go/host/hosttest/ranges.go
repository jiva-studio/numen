package hosttest

import "unicode/utf8"

// mostRead is how many bytes one ranged read of a file returns at most, as the host reads them.
const mostRead = 16000

// cutRange is the range [start, start+length) of text held to its size and cut at whole characters
// as the host cuts it: a start inside a character moves forward past it, and a character the end
// falls inside is left off, unless the range reaches the end of the text. A length of zero or less
// reaches the end.
func cutRange(text []byte, start, length int) (from, to int) {
	from, to = min(max(start, 0), len(text)), len(text)
	if length > 0 && length < len(text)-from {
		to = from + length
	}
	for from > 0 && from < to && !utf8.RuneStart(text[from]) {
		from++
	}
	if to == len(text) {
		return from, to
	}
	for pos := to - 1; pos >= from && pos > to-utf8.UTFMax; pos-- {
		if !utf8.RuneStart(text[pos]) {
			continue
		}
		if !utf8.FullRune(text[pos:to]) {
			to = pos
		}
		break
	}
	return from, to
}
