package hosttest

import "unicode/utf8"

// maxReadLength is the maximum number of bytes one ranged read of a file returns, as in the host.
const maxReadLength = 16000

// getRange returns the range [start, start+length) of text, limited to its size and truncated at
// whole characters as the host truncates it: a start inside a character moves forward past it, and a
// character the end falls inside is left out, unless the range extends to the end of the text. A
// length of zero or less extends to the end.
func getRange(text []byte, start, length int) (rangeStart, rangeEnd int) {
	rangeStart, rangeEnd = min(max(start, 0), len(text)), len(text)
	if length > 0 && length < len(text)-rangeStart {
		rangeEnd = rangeStart + length
	}
	for rangeStart > 0 && rangeStart < rangeEnd && !utf8.RuneStart(text[rangeStart]) {
		rangeStart++
	}
	if rangeEnd == len(text) {
		return rangeStart, rangeEnd
	}
	for pos := rangeEnd - 1; pos >= rangeStart && pos > rangeEnd-utf8.UTFMax; pos-- {
		if !utf8.RuneStart(text[pos]) {
			continue
		}
		if !utf8.FullRune(text[pos:rangeEnd]) {
			rangeEnd = pos
		}
		break
	}
	return rangeStart, rangeEnd
}
