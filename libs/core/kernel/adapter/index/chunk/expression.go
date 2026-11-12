package chunk

import (
	"strings"
	"unicode"
)

// Expression converts user-typed text into an FTS5 query.
// Every word is quoted into a literal and joined by implicit AND, so FTS5
// operators are unavailable: AND, OR and NEAR are searched as words. Unquoted,
// "state-function" parses as a column reference and "C++" as a syntax error.
// `growing` gives the last word a prefix mark; a finished query matches exactly.
func Expression(typed string, growing bool) string {
	var out strings.Builder
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT,STATEMENT_REMOVE reason="Grow sets capacity only; the mutated values are 1, 3 and 5, and the expression is the same at any size"
	out.Grow(2*len(typed) + 4)
	// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is -2: start is only tested against 0, so any negative marks no word"
	start := -1
	for i, r := range typed {
		if unicode.IsSpace(r) {
			if start >= 0 {
				writeQuotedWord(&out, typed[start:i])
				// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is -2: start is only tested against 0, so any negative marks no word"
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		writeQuotedWord(&out, typed[start:])
	}
	if out.Len() == 0 {
		return ""
	}
	if growing {
		out.WriteByte('*')
	}
	return out.String()
}

func writeQuotedWord(out *strings.Builder, word string) {
	// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is 1: a builder holding a word is at least 3 bytes long, so its length is never 1"
	if out.Len() > 0 {
		out.WriteByte(' ')
	}
	out.WriteByte('"')
	for i := range len(word) {
		if word[i] == '"' {
			out.WriteByte('"')
		}
		out.WriteByte(word[i])
	}
	out.WriteByte('"')
}
