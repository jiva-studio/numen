package markdown

import (
	"strings"
	"unicode/utf8"
)

// Normalised is text with every line break written as one \n.
//
// It is what a person is handed and what they hand back. An offset into a chunk
// is an offset into the file, so what Parse is given is the file as it stands.
func Normalised(text string) string {
	if !strings.ContainsRune(text, '\r') {
		return text
	}
	return breaks.Replace(text)
}

// A carriage return before a newline is one break, and a carriage return on its
// own is one as well.
var breaks = strings.NewReplacer("\r\n", "\n", "\r", "\n")

// lineEnding is what this file separates its lines with. Every break in it is
// looked at: a file whose breaks are all CRLF keeps CRLF, and a file with any
// other break in it is written with bare newlines, leaving the lines already
// written that way as they are.
func lineEnding(raw []byte) string {
	crlf := false
	for at := 0; at < len(raw); at++ {
		switch raw[at] {
		case '\n':
			if at == 0 || raw[at-1] != '\r' {
				return "\n"
			}
			crlf = true
		case '\r':
			if at+1 == len(raw) || raw[at+1] != '\n' {
				return "\n"
			}
		}
	}
	if crlf {
		return "\r\n"
	}
	return "\n"
}

// breakWidth is the bytes the line break standing at one offset takes, and zero
// where no break stands there. YAML breaks a line on a carriage return, a line
// feed, the two together, and the Unicode NEL, LS and PS.
func breakWidth(block []byte, at int) int {
	switch block[at] {
	case '\r':
		if at+1 < len(block) && block[at+1] == '\n' {
			return 2
		}
		return 1
	case '\n':
		return 1
	case 0xC2:
		if at+1 < len(block) && block[at+1] == 0x85 {
			return 2
		}
	case 0xE2:
		if at+2 < len(block) && block[at+1] == 0x80 && (block[at+2] == 0xA8 || block[at+2] == 0xA9) {
			return 3
		}
	}
	return 0
}

// endsWithBreak reports whether a block's last line is finished.
func endsWithBreak(block []byte) bool {
	for width := 1; width <= 3 && width <= len(block); width++ {
		if breakWidth(block, len(block)-width) == width {
			return true
		}
	}
	return false
}

// lineOffsets is where each line of a block begins, with the end of the block
// as a final entry, so that line n runs from offsets[n-1] to offsets[n].
func lineOffsets(block []byte) []int {
	offsets := []int{0}
	for at := 0; at < len(block); {
		width := breakWidth(block, at)
		if width == 0 {
			at++
			continue
		}
		at += width
		offsets = append(offsets, at)
	}
	if offsets[len(offsets)-1] != len(block) {
		offsets = append(offsets, len(block))
	}
	return offsets
}

// columnOffset is where the column the parser reports on one line stands in the
// block, in bytes. The parser counts a column in characters, and a column past
// the end of its line stands at the end of it.
func columnOffset(block []byte, lines []int, line, column int) int {
	at, end := lines[line-1], lines[line]
	for count := 1; count < column && at < end; count++ {
		_, width := utf8.DecodeRune(block[at:end])
		at += width
	}
	return at
}
