package chunk_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestFindLexicalLimitsWithinPathPrefix(t *testing.T) {
	db := searchFixture(t)
	queries := db.ChunkQueries()

	for _, kinds := range [][]domain.SourceKind{nil, {domain.KindNote, domain.KindFile}} {
		got, err := queries.FindLexical(t.Context(), vaultA, sourceport.LexicalQuery{Text: "zebra", Kinds: kinds, Limit: 1, IsGrowing: false, PathPrefix: "notes/"})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(getSources(got), []string{"notes/zebra.md"}) {
			t.Errorf("kinds %v: one passage under notes/ is %v, want the note although the book ranks higher", kinds, getSources(got))
		}
	}

	books, err := queries.FindLexical(t.Context(), vaultA, sourceport.LexicalQuery{Text: "zebra", Kinds: nil, Limit: 10, IsGrowing: false, PathPrefix: "lib/zebra"})
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 2 || books[0].Source != "lib/zebra.epub" {
		t.Errorf("a prefix that ends inside a file name found %v, want the two chunks of the book", getSources(books))
	}

	none, err := queries.FindLexical(t.Context(), vaultA, sourceport.LexicalQuery{Text: "zebra", Kinds: nil, Limit: 10, IsGrowing: false, PathPrefix: "elsewhere/"})
	if err != nil || len(none) != 0 {
		t.Errorf("a prefix nothing starts with found %v, %v", getSources(none), err)
	}
}

func TestAPathPrefixIsNotAPattern(t *testing.T) {
	db := openIndex(t)
	source{path: "a_b/one.md", kind: "note", large: "zebra", smalls: []string{"zebra"}}.save(t, db)
	source{path: "aXb/two.md", kind: "note", large: "zebra", smalls: []string{"zebra"}}.save(t, db)
	source{path: "100%/three.md", kind: "note", large: "zebra", smalls: []string{"zebra"}}.save(t, db)
	source{path: "1000/four.md", kind: "note", large: "zebra", smalls: []string{"zebra"}}.save(t, db)
	queries := db.ChunkQueries()

	for prefix, want := range map[string]string{"a_b/": "a_b/one.md", "100%": "100%/three.md"} {
		got, err := queries.FindLexical(t.Context(), vaultA, sourceport.LexicalQuery{Text: "zebra", Kinds: nil, Limit: 10, IsGrowing: false, PathPrefix: prefix})
		if err != nil || !slices.Equal(getSources(got), []string{want}) {
			t.Errorf("prefix %q found %v, %v, want only %s", prefix, getSources(got), err, want)
		}
	}
}

func TestGetNamedPassagesLimitsWithinPathPrefix(t *testing.T) {
	db := openIndex(t)
	source{
		path: "lib/guide.epub", kind: "file", producer: "ocr", large: "text", opens: []string{"Gardening"}, smalls: []string{"x"},
	}.save(t, db)
	source{path: "notes/Gardening.md", kind: "note", large: "text", opens: []string{"Gardening notes"}, smalls: []string{"y"}}.save(t, db)
	queries := db.ChunkQueries()

	for _, kinds := range [][]domain.SourceKind{nil, {domain.KindNote, domain.KindFile}} {
		got, err := queries.GetNamedPassages(t.Context(), vaultA, sourceport.LexicalQuery{Text: "gardening", Kinds: kinds, Limit: 1, IsGrowing: false, PathPrefix: "notes/"})
		if err != nil || !slices.Equal(getSources(got), []string{"notes/Gardening.md"}) {
			t.Errorf("kinds %v: the section under notes/ is %v, %v", kinds, getSources(got), err)
		}
	}
}

func TestFindNearestWithinPathPrefixReturnsOnlyPrefixChunks(t *testing.T) {
	db := openIndex(t)
	for i := range 12 {
		path := fmt.Sprintf("lib/near-%02d.epub", i)
		source{path: path, kind: "file", producer: "ocr", hash: "h-" + path, large: "large " + path, smalls: []string{"first " + path, "second " + path}}.save(t, db)
		embed(t, db, vaultA, path, 0xF0)
	}
	source{path: "notes/far.md", kind: "note", hash: "h-far", large: "large far", smalls: []string{"first far"}}.save(t, db)
	embed(t, db, vaultA, "notes/far.md", 0x0F)

	got, err := db.ChunkQueries().FindNearest(t.Context(), vaultA, sourceport.NearestQuery{
		Recipe: recipe, Vector: seedVector(0xF0), Limit: 1, Floor: -1, PathPrefix: "notes/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(getSources(got), []string{"notes/far.md"}) {
		t.Errorf("one passage under notes/ is %v, want the note although 24 nearer chunks lie elsewhere", getSources(got))
	}

	kinded, err := db.ChunkQueries().FindNearest(t.Context(), vaultA, sourceport.NearestQuery{
		Recipe: recipe, Vector: seedVector(0xF0), Kinds: []domain.SourceKind{domain.KindNote}, Limit: 1, Floor: -1, PathPrefix: "notes/",
	})
	if err != nil || len(kinded) != 1 {
		t.Errorf("a prefix with a kind found %d, %v", len(kinded), err)
	}

	none, err := db.ChunkQueries().FindNearest(t.Context(), vaultA, sourceport.NearestQuery{
		Recipe: recipe, Vector: seedVector(0xF0), Limit: 3, Floor: -1, PathPrefix: "elsewhere/",
	})
	if err != nil || len(none) != 0 {
		t.Errorf("a prefix nothing starts with found %v, %v", getSources(none), err)
	}
}
