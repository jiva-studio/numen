package chunk

import (
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// typedSeeds are the things a person types into a search box: words, the
// punctuation FTS5 reads as syntax, its own operators, a quote left open, a
// column reference, a phrase, other scripts, and nothing at all.
var typedSeeds = []string{
	"entropy",
	"entropy shannon",
	"state-function",
	"C++",
	`say"hello`,
	`"entropy`,
	`"""`,
	"a:b",
	"one (two)",
	"NEAR(a b, 2)",
	"a OR b AND NOT c",
	"^entropy",
	"*",
	"наставник",
	"शब्द",
	"a\x00b",
	"\xff\xfe",
	strings.Repeat("a ", 200),
	"   ",
	"",
}

// mostAsked is how much of what was typed is put to a database.
const mostAsked = 256

// Nothing a person can type reaches FTS5 as syntax. Whatever is typed, the
// expression it is turned into is a query FTS5 accepts, and the words it asks
// for are the words that were typed and no others.
//
// What is typed is a stranger's: it comes off a search box and, through the
// wire, off anything that can address the application.
func FuzzExpression(f *testing.F) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		f.Fatal(err)
	}
	f.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE VIRTUAL TABLE said USING fts5(text)`); err != nil {
		f.Skipf("this build of SQLite has no FTS5: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO said(text) VALUES ('entropy is a state function')`); err != nil {
		f.Fatal(err)
	}

	for _, typed := range typedSeeds {
		f.Add(typed, true)
		f.Add(typed, false)
	}

	f.Fuzz(func(t *testing.T, typed string, growing bool) {
		// A NUL ends a string on its way into SQLite, so what the driver would
		// ask is not what was written. Nothing types one; the escaping is still
		// held to the round trip below.
		//
		// FTS5 costs more than linearly in the number of words asked for — ten
		// thousand of them take a third of a second — so what is put to the
		// database is what a search box could hold. The round trip below is
		// asked of every length.
		expression := Expression(typed, growing)
		if len(typed) <= mostAsked && !strings.ContainsRune(expression, 0) {
			var count int
			row := db.QueryRow(`SELECT count(*) FROM said WHERE text MATCH ?`, expression)
			// An empty expression is not a query, and no search is made.
			if expression != "" {
				if err := row.Scan(&count); err != nil {
					t.Fatalf("%q was asked as %q, which FTS5 refused: %v", typed, expression, err)
				}
			}
		}

		fields := strings.Fields(typed)
		if len(fields) == 0 {
			if expression != "" {
				t.Fatalf("%q holds no word and was asked as %q", typed, expression)
			}
			return
		}
		if asked := words(t, expression, growing); len(asked) != len(fields) {
			t.Fatalf("%q holds %d words and was asked as %q, which holds %d",
				typed, len(fields), expression, len(asked))
		} else {
			for i, word := range asked {
				if word != fields[i] {
					t.Fatalf("%q asks for %q where %q was typed", expression, word, fields[i])
				}
			}
		}
	})
}

// words reads the literals of an expression back into the words they stand for,
// failing unless every one of them is a closed FTS5 string and the whole
// expression is nothing but those literals with a single space between them.
func words(t *testing.T, expression string, growing bool) []string {
	t.Helper()
	if growing {
		last, cut := strings.CutSuffix(expression, "*")
		if !cut {
			t.Fatalf("%q asks for a word still being typed and carries no prefix mark", expression)
		}
		expression = last
	}

	var out []string
	rest := expression
	for {
		if !strings.HasPrefix(rest, `"`) {
			t.Fatalf("%q holds %q, which is not a literal", expression, rest)
		}
		var word strings.Builder
		rest = rest[1:]
		for {
			at := strings.IndexByte(rest, '"')
			if at < 0 {
				t.Fatalf("%q holds a literal that never closes", expression)
			}
			word.WriteString(rest[:at])
			// A doubled quote is one quote of the word; a single one closes it.
			if strings.HasPrefix(rest[at+1:], `"`) {
				word.WriteByte('"')
				rest = rest[at+2:]
				continue
			}
			rest = rest[at+1:]
			break
		}
		out = append(out, word.String())
		if rest == "" {
			return out
		}
		after, joined := strings.CutPrefix(rest, " ")
		if !joined {
			t.Fatalf("%q joins two literals with %q", expression, rest)
		}
		rest = after
	}
}
