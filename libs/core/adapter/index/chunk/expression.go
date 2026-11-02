package chunk

import (
	"strings"
	"unicode"

	"github.com/jiva-studio/numen/libs/core/internal/wordform"
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

// Expressions converts user-typed text into the FTS5 queries that find it, from
// the strictest to the loosest. A search runs them in this order and lists the
// chunks of an earlier one first:
//
//  1. every word exactly;
//  2. the last word as a prefix of a word of the text (growing only);
//  3. every word of four letters or more by its stem, as a prefix, so a word
//     is found in another form: "pizda" finds "pizde", "knigi" finds "knigu"
//     (growing only).
//
// A finished query (growing false) matches exactly, as Expression does. A query
// equal to an earlier one is left out.
func Expressions(typed string, growing bool) []string {
	words := strings.Fields(typed)
	if len(words) == 0 {
		return nil
	}
	exact := Expression(typed, false)
	out := []string{exact}
	if !growing {
		return out
	}
	if prefix := Expression(typed, true); prefix != exact {
		out = append(out, prefix)
	}

	var stemmed strings.Builder
	hasStem := false
	for _, w := range words {
		stem := wordform.GetStem(w)
		if stem == "" {
			writeQuoted(&stemmed, w)
			continue
		}
		hasStem = true
		writeQuoted(&stemmed, stem)
		stemmed.WriteByte('*')
	}
	if stems := stemmed.String(); hasStem && stems != out[len(out)-1] {
		out = append(out, stems)
	}
	return out
}
