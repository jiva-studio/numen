package note

import "strings"

// ftsExpression turns what a person typed into an FTS5 query.
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
func ftsExpression(typed string) string {
	fields := strings.Fields(typed)
	if len(fields) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(fields))
	for _, field := range fields {
		// Doubling is how a quote is escaped inside an FTS5 string.
		quoted = append(quoted, `"`+strings.ReplaceAll(field, `"`, `""`)+`"`)
	}
	return strings.Join(quoted, " ")
}
