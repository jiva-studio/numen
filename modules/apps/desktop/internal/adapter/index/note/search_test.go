package note

import "testing"

func TestOrdinaryWordsAreNotFTSSyntax(t *testing.T) {
	// Each of these is a real thing to search for, and each of them is a syntax
	// error if it reaches FTS5 unquoted: a hyphen is read as a column reference,
	// a plus as an operator, a quote as the start of a string.
	for _, typed := range []string{"state-function", "C++", `"entropy`, "a:b", "one (two)"} {
		got := ftsExpression(typed)
		if got == "" {
			t.Errorf("%q produced no expression", typed)
		}
		if got[0] != '"' {
			t.Errorf("%q produced %q, which is not quoted", typed, got)
		}
	}
}

func TestWordsAreCombinedWithImplicitAnd(t *testing.T) {
	if got, want := ftsExpression("entropy shannon"), `"entropy" "shannon"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEmbeddedQuotesAreDoubled(t *testing.T) {
	if got, want := ftsExpression(`say"hello`), `"say""hello"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBlankInputProducesNoExpression(t *testing.T) {
	// An empty expression is not a query that matches everything; the caller
	// returns nothing rather than asking FTS5 to parse "".
	for _, typed := range []string{"", "   ", "\t\n"} {
		if got := ftsExpression(typed); got != "" {
			t.Errorf("%q produced %q", typed, got)
		}
	}
}
