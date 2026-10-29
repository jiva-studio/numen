package markdown

import (
	"strings"
	"unicode/utf8"
)

// Normalise is text with every line break written as one \n.
//
// It is the form clients receive and send back. An offset into a chunk is an
// offset into the file, so the text passed to Parse is the current file.
func Normalise(text string) string {
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
	for pos := 0; pos < len(raw); pos++ {
		switch raw[pos] {
		case '\n':
			if pos == 0 || raw[pos-1] != '\r' {
				return "\n"
			}
			crlf = true
		case '\r':
			if pos+1 == len(raw) || raw[pos+1] != '\n' {
				return "\n"
			}
		}
	}
	if crlf {
		return "\r\n"
	}
	return "\n"
}

// breakWidth returns the width in bytes of the line break at an offset, and
// zero when no break is there. YAML breaks a line on a carriage return, a line
// feed, the two together, and the Unicode NEL, LS and PS.
func breakWidth(block []byte, offset int) int {
	switch block[offset] {
	case '\r':
		if offset+1 < len(block) && block[offset+1] == '\n' {
			return 2
		}
		return 1
	case '\n':
		return 1
	case 0xC2:
		if offset+1 < len(block) && block[offset+1] == 0x85 {
			return 2
		}
	case 0xE2:
		if offset+2 < len(block) && block[offset+1] == 0x80 && (block[offset+2] == 0xA8 || block[offset+2] == 0xA9) {
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

// lineOffsets returns the start offset of each line of a block, with the end of
// the block as a final entry, so line n runs from offsets[n-1] to offsets[n].
func lineOffsets(block []byte) []int {
	offsets := []int{0}
	for pos := 0; pos < len(block); {
		width := breakWidth(block, pos)
		if width == 0 {
			pos++
			continue
		}
		pos += width
		offsets = append(offsets, pos)
	}
	if offsets[len(offsets)-1] != len(block) {
		offsets = append(offsets, len(block))
	}
	return offsets
}

// block, in bytes. The parser counts a column in characters, and a column past
func columnOffset(block []byte, lines []int, line, column int) int {
	offset, end := lines[line-1], lines[line]
	for count := 1; count < column && offset < end; count++ {
		_, width := utf8.DecodeRune(block[offset:end])
		offset += width
	}
	return offset
}

func getIndent(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// offsetLine returns the 1-based line that begins at or after a byte offset, or
// len(lines) when no line starts that far into the block.
func offsetLine(lines []int, offset int) int {
	for i, lineStart := range lines {
		if lineStart >= offset {
			return i + 1
		}
	}
	return len(lines)
}
