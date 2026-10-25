package chunk

import (
	"strings"
	"unicode"
)

// Expression turns what a person typed into an FTS5 query.
//
// The value is already bound as a parameter, so this is not about injection: it
// is that FTS5 parses the bound string as an expression of its own. Left alone,
// ordinary words fail — "state-function" is read as a column reference, "C++" as
// a syntax error, a stray quote as an unterminated string — and the person
// searching gets a database error.
//
// Every word is therefore quoted into a literal, and the words are joined by
// implicit AND. The cost is that FTS5's own operators are not available to the
// user: searching for AND, OR or NEAR finds those words. That is the right trade
// for a search box — a query language is a decision to make deliberately, not
// something to leak because of how a string is passed along.
//
// `growing` says the last word may still be being typed, and it then carries a
// prefix mark. A question that is finished is asked exactly.
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
