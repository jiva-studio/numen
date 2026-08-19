package chunk

import "strings"

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
// The last word carries a prefix mark, because a search box is typed into: the
// index holds whole words, and a word still being typed matches none of them.
// It is what lets "наставник" reach "наставника", and a finished word is a
// prefix of itself, so nothing is lost by it.
func Expression(typed string) string {
	fields := strings.Fields(typed)
	if len(fields) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(fields))
	for _, field := range fields {
		// Doubling is how a quote is escaped inside an FTS5 string.
		quoted = append(quoted, `"`+strings.ReplaceAll(field, `"`, `""`)+`"`)
	}
	quoted[len(quoted)-1] += "*"
	return strings.Join(quoted, " ")
}
