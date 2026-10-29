package chunk

import (
	"strings"
	"unicode"
)

// Expression converts user-typed text into an FTS5 query.
//
// The value is bound as a parameter, so injection is not the concern: FTS5 parses
// the bound string as an expression of its own. Unprocessed, ordinary input
// fails: "state-function" is parsed as a column reference, "C++" as a syntax
// error, and a stray quote as an unterminated string, so the search returns a
// database error.
//
// Every word is quoted into a literal, and the words are joined by implicit AND.
// FTS5 operators are therefore unavailable: searching for AND, OR or NEAR finds
// those words. A query language is an explicit feature, not a side effect of how
// a string is passed.
//
// `growing` marks the last word as possibly incomplete, and it then receives a
// prefix mark. A finished query matches exactly.
func Expression(typed string, growing bool) string {
	var out strings.Builder
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT,STATEMENT_REMOVE reason="Grow sets capacity only; the mutated values are 1, 3 and 5, and the expression is the same at any size"
	out.Grow(2*len(typed) + 4)
	// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is -2: start is only tested against 0, so any negative marks no word"
	start := -1
	for i, r := range typed {
		if unicode.IsSpace(r) {
			if start >= 0 {
				writeQuoted(&out, typed[start:i])
				// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is -2: start is only tested against 0, so any negative marks no word"
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		writeQuoted(&out, typed[start:])
	}
	if out.Len() == 0 {
		return ""
	}
	if growing {
		out.WriteByte('*')
	}
	return out.String()
}

// writeQuoted appends word as an FTS5 string literal, separated from the
// previous word by a space. Doubling is how a quote is escaped inside an FTS5
// string.
func writeQuoted(out *strings.Builder, word string) {
	// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is 1: a builder holding a word is at least 3 bytes long, so its length is never 1"
	if out.Len() > 0 {
		out.WriteByte(' ')
	}
	out.WriteByte('"')
	for i := 0; i < len(word); i++ {
		if word[i] == '"' {
			out.WriteByte('"')
		}
		out.WriteByte(word[i])
	}
	out.WriteByte('"')
}
