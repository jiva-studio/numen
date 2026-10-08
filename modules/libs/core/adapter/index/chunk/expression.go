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
	out.Grow(2*len(typed) + 4)
	start := -1
	for i, r := range typed {
		if unicode.IsSpace(r) {
			if start >= 0 {
				writeQuoted(&out, typed[start:i])
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
