package hosttest

import "unicode/utf8"

// maxReadLength is the maximum number of bytes one ranged read of a file returns, as in the host.
const maxReadLength = 16000

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
