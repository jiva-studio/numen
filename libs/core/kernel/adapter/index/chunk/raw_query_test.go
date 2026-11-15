package chunk_test

import (
	"errors"
	"slices"
	"testing"

	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
)

func TestFindLexicalMatchesARawPhraseQueryAsAPhrase(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/ordered.epub", kind: "file", large: "one", smalls: []string{"the gray heron waits"}}.save(t, db)
	source{path: "lib/apart.epub", kind: "file", large: "two", smalls: []string{"a heron, gray and old"}}.save(t, db)

	raw, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: `"gray heron"`, Limit: 10, IsRaw: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := getSources(raw); !slices.Equal(got, []string{"lib/ordered.epub"}) {
		t.Errorf("a raw phrase found %v, want only the book with the words in order", got)
	}

	quoted, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: `"gray heron"`, Limit: 10})
	if err != nil || len(quoted) != 2 {
		t.Errorf("the same text not raw found %v, %v, want both books by their words", getSources(quoted), err)
	}
}

func TestFindLexicalMatchesARawPrefixQuery(t *testing.T) {
	db := searchFixture(t)

	raw, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "strip*", Limit: 10, IsRaw: true})
	if err != nil || len(raw) != 1 || raw[0].Path != "lib/zebra.epub" {
		t.Errorf("a raw prefix found %v, %v, want the book with stripes", getSources(raw), err)
	}

	quoted, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "strip*", Limit: 10})
	if err != nil || len(quoted) != 0 {
		t.Errorf("the same text not raw found %v, %v, want nothing", getSources(quoted), err)
	}
}

func TestFindLexicalQuotesTheWordsOfAQueryThatIsNotRaw(t *testing.T) {
	db := openIndex(t)
	source{path: "lib/near.epub", kind: "file", large: "one", smalls: []string{"the zebra drinks near water"}}.save(t, db)
	source{path: "lib/by.epub", kind: "file", large: "two", smalls: []string{"the zebra by the water"}}.save(t, db)

	quoted, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra NEAR water", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got := getSources(quoted); !slices.Equal(got, []string{"lib/near.epub"}) {
		t.Errorf("a query that is not raw found %v, want only the book with the word near", got)
	}

	raw, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "NEAR(zebra water, 3)", Limit: 10, IsRaw: true})
	if err != nil || len(raw) != 2 {
		t.Errorf("a raw NEAR query found %v, %v, want both books", getSources(raw), err)
	}
}

func TestFindLexicalReturnsErrInvalidWordsForARawQueryFTS5CannotParse(t *testing.T) {
	db := searchFixture(t)

	for _, text := range []string{`"zebra`, "zebra AND", "NEAR(zebra"} {
		_, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: text, Limit: 10, IsRaw: true})
		if !errors.Is(err, sourcedomain.ErrInvalidWords) {
			t.Errorf("the raw query %q returned %v, want ErrInvalidWords", text, err)
		}
	}
}

func TestFindLexicalReturnsAPositiveWordScoreBestFirst(t *testing.T) {
	db := searchFixture(t)

	found, err := db.ChunkSearch().FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Limit: 10})
	if err != nil || len(found) == 0 {
		t.Fatalf("the words found %v, %v", getSources(found), err)
	}
	for i, p := range found {
		if p.WordScore == nil || *p.WordScore <= 0 {
			t.Errorf("the hit in %s has the word score %v, want a positive one", p.Path, p.WordScore)
		}
		if i > 0 && found[i-1].WordScore != nil && p.WordScore != nil && *p.WordScore > *found[i-1].WordScore {
			t.Errorf("the hits are best first, and %s scores above the one before it", p.Path)
		}
	}
}
