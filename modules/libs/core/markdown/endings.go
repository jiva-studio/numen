package markdown

import "strings"

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
