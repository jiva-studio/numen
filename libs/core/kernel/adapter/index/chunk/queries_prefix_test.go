package chunk_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestFindLexicalLimitsWithinPathPrefix(t *testing.T) {
	db := searchFixture(t)
	queries := queriesOf(db)

	for _, kinds := range [][]domain.SourceKind{nil, {domain.KindNote, domain.KindFile}} {
		got, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: kinds, Limit: 1, IsGrowing: false, PathPrefix: "notes/"})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(getSources(got), []string{"notes/zebra.md"}) {
			t.Errorf("kinds %v: one passage under notes/ is %v, want the note although the book ranks higher", kinds, getSources(got))
		}
	}

	books, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: nil, Limit: 10, IsGrowing: false, PathPrefix: "lib/zebra"})
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 2 || books[0].Path != "lib/zebra.epub" {
		t.Errorf("a prefix that ends inside a file name found %v, want the two chunks of the book", getSources(books))
	}

	none, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: nil, Limit: 10, IsGrowing: false, PathPrefix: "elsewhere/"})
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
	queries := queriesOf(db)

	for prefix, want := range map[string]string{"a_b/": "a_b/one.md", "100%": "100%/three.md"} {
		got, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "zebra", Kinds: nil, Limit: 10, IsGrowing: false, PathPrefix: prefix})
		if err != nil || !slices.Equal(getSources(got), []string{want}) {
			t.Errorf("prefix %q found %v, %v, want only %s", prefix, getSources(got), err, want)
		}
	}
}

func TestGetNamedSectionsLimitsWithinPathPrefix(t *testing.T) {
	db := openIndex(t)
	source{
		path: "lib/guide.epub", kind: "file", producer: "ocr", large: "text", opens: []string{"Gardening"}, smalls: []string{"x"},
	}.save(t, db)
	source{path: "notes/Gardening.md", kind: "note", large: "text", opens: []string{"Gardening notes"}, smalls: []string{"y"}}.save(t, db)
	queries := queriesOf(db)

	for _, kinds := range [][]domain.SourceKind{nil, {domain.KindNote, domain.KindFile}} {
		got, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "gardening", Kinds: kinds, Limit: 1, IsGrowing: false, PathPrefix: "notes/"})
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

	got, err := queriesOf(db).FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{
		Recipe: recipe, Vector: seedVector(0xF0), Limit: 1, Floor: -1, PathPrefix: "notes/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(getSources(got), []string{"notes/far.md"}) {
		t.Errorf("one passage under notes/ is %v, want the note although 24 nearer chunks lie elsewhere", getSources(got))
	}

	kinded, err := queriesOf(db).FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{
		Recipe: recipe, Vector: seedVector(0xF0), Kinds: []domain.SourceKind{domain.KindNote}, Limit: 1, Floor: -1, PathPrefix: "notes/",
	})
	if err != nil || len(kinded) != 1 {
		t.Errorf("a prefix with a kind found %d, %v", len(kinded), err)
	}

	none, err := queriesOf(db).FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{
		Recipe: recipe, Vector: seedVector(0xF0), Limit: 3, Floor: -1, PathPrefix: "elsewhere/",
	})
	if err != nil || len(none) != 0 {
		t.Errorf("a prefix nothing starts with found %v, %v", getSources(none), err)
	}
}

// A Linux file name need not be UTF-8, and its bytes after the prefix may sort above every
// character: the range on the path is bounded by bytes, not by the largest character.
func TestFindNearestWithinPathPrefixFindsAPathThatIsNotUTF8(t *testing.T) {
	db := openIndex(t)
	for _, path := range []string{"notes/\xff.md", "\xff\xff.md", "notes0.md"} {
		source{path: path, kind: "note", hash: "h-" + path, large: "large " + path, smalls: []string{"first " + path}}.save(t, db)
		embed(t, db, vaultA, path, 0xF0)
	}

	for prefix, want := range map[string]string{"notes/": "notes/\xff.md", "\xff": "\xff\xff.md"} {
		got, err := queriesOf(db).FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{
			Recipe: recipe, Vector: seedVector(0xF0), Limit: 10, Floor: -1, PathPrefix: prefix,
		})
		if err != nil {
			t.Fatal(err)
		}
		if sources := slices.Compact(getSources(got)); !slices.Equal(sources, []string{want}) {
			t.Errorf("under %q the passages are of %q, want only %q", prefix, sources, want)
		}
	}
}

// The two forms of the invented shelf "Bücher": ü as one character (NFC), and as u followed by a
// combining diaeresis (NFD), as some file systems store it.
const (
	shelfComposed   = "Bücher"
	shelfDecomposed = "Bücher"
)

func openShelfIndex(t *testing.T, shelf string) (*index.DB, string) {
	t.Helper()
	db := openIndex(t)
	book := shelf + "/Teebuch.md"
	for _, path := range []string{book, "library/Tea.md"} {
		source{path: path, kind: "note", hash: "h-" + path, large: "tea", opens: []string{"Tea"}, smalls: []string{"tea"}}.save(t, db)
		embed(t, db, vaultA, path, 0xF0)
	}
	return db, book
}

func TestAPathPrefixMatchesTheFolderInEitherNormalForm(t *testing.T) {
	for _, stored := range []string{shelfDecomposed, shelfComposed} {
		db, book := openShelfIndex(t, stored)
		queries := queriesOf(db)
		for _, prefix := range []string{shelfComposed + "/", shelfDecomposed + "/"} {
			lexical, err := queries.FindLexical(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "tea", Limit: 10, PathPrefix: prefix})
			if err != nil || !slices.Equal(slices.Compact(getSources(lexical)), []string{book}) {
				t.Errorf("stored %+q, prefix %+q: FindLexical found %+q, %v, want only %+q", stored, prefix, getSources(lexical), err, book)
			}
			named, err := queries.GetNamedSections(t.Context(), vaultA, sourcedomain.LexicalQuery{Text: "tea", Limit: 10, PathPrefix: prefix})
			if err != nil || !slices.Equal(slices.Compact(getSources(named)), []string{book}) {
				t.Errorf("stored %+q, prefix %+q: GetNamedSections found %+q, %v, want only %+q", stored, prefix, getSources(named), err, book)
			}
			nearest, err := queries.FindNearest(t.Context(), vaultA, sourcedomain.NearestQuery{
				Recipe: recipe, Vector: seedVector(0xF0), Limit: 10, Floor: -1, PathPrefix: prefix,
			})
			if err != nil || !slices.Equal(slices.Compact(getSources(nearest)), []string{book}) {
				t.Errorf("stored %+q, prefix %+q: FindNearest found %+q, %v, want only %+q", stored, prefix, getSources(nearest), err, book)
			}
		}
	}
}
