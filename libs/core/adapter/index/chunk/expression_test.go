package chunk

import (
	"strings"
	"testing"
)

func TestOrdinaryWordsAreNotFTSSyntax(t *testing.T) {
	// Each of these is a legitimate search term, and each is a syntax error when
	// passed to FTS5 unquoted: a hyphen is read as a column reference, a plus as an
	// operator, a quote as the start of a string.
	for _, input := range []string{"state-function", "C++", `"entropy`, "a:b", "one (two)"} {
		got := Expression(input, true)
		if got == "" {
			t.Errorf("%q produced no expression", input)
		}
		if got[0] != '"' {
			t.Errorf("%q produced %q, which is not quoted", input, got)
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
	for _, input := range []string{"", "   ", "\t\n"} {
		if got := Expression(input, true); got != "" {
			t.Errorf("%q produced %q", input, got)
		}
	}
}

func TestExpression_ExactTermsWhenNotGrowing(t *testing.T) {
	// A search by words is what answers a question spelled precisely. An agent
	// asking whether a word is in the vault is asking about that word.
	if got, want := Expression("наставник", false), `"наставник"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := Expression("entro mixing", false), `"entro" "mixing"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func expressionReference(typed string, growing bool) string {
	fields := strings.Fields(typed)
	if len(fields) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(fields))
	for _, field := range fields {
		quoted = append(quoted, `"`+strings.ReplaceAll(field, `"`, `""`)+`"`)
	}
	if growing {
		quoted[len(quoted)-1] += "*"
	}
	return strings.Join(quoted, " ")
}

var expressionInputs = []string{
	"", " ", "\t\n ", "a", "entropy", "entropy shannon", "  entropy   shannon  ",
	"state-function", "C++", `"entropy`, `say"hello`, `""`, `"`, `a "b" c`,
	"наставник", "энтропия Шеннона", "日本語 検索", "a\u00a0b", "a\u2003b\u3000c",
	"a\x85b", "bad\xffbyte x\xc3", "emoji 😀 test", "one (two)", "a:b",
	"\u200bzero", "tab\tsep\nnl\rcr\vvt\ffeed",
}

func TestExpressionMatchesReference(t *testing.T) {
	for _, input := range expressionInputs {
		for _, growing := range []bool{false, true} {
			if got, want := Expression(input, growing), expressionReference(input, growing); got != want {
				t.Errorf("Expression(%q, %v) = %q, want %q", input, growing, got, want)
			}
		}
	}
}

func BenchmarkExpression(b *testing.B) {
	cases := []struct{ name, typed string }{
		{"one-word", "entropy"},
		{"three-words", "entropy shannon mixing"},
		{"quotes", `say "hello" to "world"`},
		{"unicode", "энтропия Шеннона наставник"},
		{"long", strings.Repeat("thermodynamic state-function ", 8)},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = Expression(c.typed, true)
			}
		})
	}
}
