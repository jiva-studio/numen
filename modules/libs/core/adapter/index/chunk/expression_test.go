package chunk

import "testing"

func TestOrdinaryWordsAreNotFTSSyntax(t *testing.T) {
	// Each of these is a real thing to search for, and each of them is a syntax
	// error if it reaches FTS5 unquoted: a hyphen is read as a column reference,
	// a plus as an operator, a quote as the start of a string.
	for _, typed := range []string{"state-function", "C++", `"entropy`, "a:b", "one (two)"} {
		got := Expression(typed, true)
		if got == "" {
			t.Errorf("%q produced no expression", typed)
		}
		if got[0] != '"' {
			t.Errorf("%q produced %q, which is not quoted", typed, got)
		}
	}
}

func TestWordsAreCombinedWithImplicitAnd(t *testing.T) {
	if got, want := Expression("entropy shannon", true), `"entropy" "shannon"*`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTheLastWordIsMatchedOnItsPrefix(t *testing.T) {
	// The index holds whole words, and a word still being typed is a prefix of
	// the one in the text.
	if got, want := Expression("наставник", true), `"наставник"*`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// Only the last one: the words before it are finished.
	if got, want := Expression("entro mixing", true), `"entro" "mixing"*`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEmbeddedQuotesAreDoubled(t *testing.T) {
	if got, want := Expression(`say"hello`, true), `"say""hello"*`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBlankInputProducesNoExpression(t *testing.T) {
	// An empty expression is not a query that matches everything; the caller
	// answers with nothing.
	for _, typed := range []string{"", "   ", "\t\n"} {
		if got := Expression(typed, true); got != "" {
			t.Errorf("%q produced %q", typed, got)
		}
	}
}

func TestAFinishedQuestionIsAskedExactly(t *testing.T) {
	// A search by words is what answers a question spelled precisely. An agent
	// asking whether a word is in the vault is asking about that word.
	if got, want := Expression("наставник", false), `"наставник"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := Expression("entro mixing", false), `"entro" "mixing"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
