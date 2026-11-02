package index

import (
	"slices"
	"strconv"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// saveNoteText stores a note of one chunk holding the text, with no vector.
func saveNoteText(t *testing.T, db *DB, vault domain.Vault, path, text string) {
	t.Helper()
	ctx := t.Context()
	if err := db.Chunks().SaveSource(ctx, vault.ID, chunk.Source{Path: path, Kind: "note", Size: int64(len(text)), MTime: 1}); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().ReplaceChunks(ctx, vault.ID, "note", path, []chunk.Chunk{{Start: 0, Length: len(text), Text: text, Small: []chunk.Chunk{{Start: 0, Length: len(text), Text: text}}}}); err != nil {
		t.Fatal(err)
	}
}

// foundBy is the paths a typed query finds, best first.
func foundBy(t *testing.T, db *DB, typed string, growing bool) []string {
	t.Helper()
	found, err := db.ChunkQueries().FindLexical(t.Context(), getFirst().ID, typed, nil, 10, growing)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, p := range found {
		paths = append(paths, p.Source)
	}
	return paths
}

func TestTheExactFormRanksFirstAndAnotherFormIsFound(t *testing.T) {
	db := openDB(t)
	saveNoteText(t, db, getFirst(), "n/a.md", "kuvshin stoit na vode u okna")
	saveNoteText(t, db, getFirst(), "n/b.md", "kholodnaya voda iz kolodtsa")

	for _, c := range []struct {
		typed string
		want  []string
	}{
		{"voda", []string{"n/b.md", "n/a.md"}},
		{"vode", []string{"n/a.md", "n/b.md"}},
		{"vodu", []string{"n/b.md", "n/a.md"}},
		{"vod", []string{"n/b.md", "n/a.md"}},
	} {
		got := foundBy(t, db, c.typed, true)
		if c.typed == "vodu" || c.typed == "vod" {
			slices.Sort(got)
			slices.Sort(c.want)
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%q found %v, want %v", c.typed, got, c.want)
		}
	}
}

func TestAFinishedQueryStaysExact(t *testing.T) {
	db := openDB(t)
	saveNoteText(t, db, getFirst(), "n/a.md", "stoit na vode")
	if got := foundBy(t, db, "voda", false); len(got) != 0 {
		t.Errorf("a finished query found %v in another form", got)
	}
}

func TestAShortWordIsAPrefixAndNotAStem(t *testing.T) {
	db := openDB(t)
	saveNoteText(t, db, getFirst(), "n/a.md", "dom na holme")
	saveNoteText(t, db, getFirst(), "n/b.md", "domik v lesu")
	saveNoteText(t, db, getFirst(), "n/c.md", "dym nad truboy")

	if got, want := foundBy(t, db, "dom", true), []string{"n/a.md", "n/b.md"}; !slices.Equal(got, want) {
		t.Errorf("found %v, want %v", got, want)
	}
	if got := foundBy(t, db, "dam", true); len(got) != 0 {
		t.Errorf("a different three-letter word found %v", got)
	}
}

func TestWordFormsInCyrillicAndWithDiacritics(t *testing.T) {
	db := openDB(t)
	saveNoteText(t, db, getFirst(), "n/a.md", "Лежит на столе книге")
	saveNoteText(t, db, getFirst(), "n/b.md", "Моя КНИГА большая")
	saveNoteText(t, db, getFirst(), "n/c.md", "hare Kṛṣṇa hare")

	if got, want := foundBy(t, db, "книгу", true), []string{"n/a.md", "n/b.md"}; len(got) != 2 {
		t.Errorf("found %v, want both of %v", got, want)
	}
	if got, want := foundBy(t, db, "книга", true), "n/b.md"; len(got) != 2 || got[0] != want {
		t.Errorf("found %v, want %s first", got, want)
	}
	if got := foundBy(t, db, "krsna", true); !slices.Equal(got, []string{"n/c.md"}) {
		t.Errorf("krsna found %v", got)
	}
	if got := foundBy(t, db, "KRSNE", true); !slices.Equal(got, []string{"n/c.md"}) {
		t.Errorf("KRSNE found %v", got)
	}
}

func TestAFreshNoteIsSearchableByWordsBeforeItsVector(t *testing.T) {
	db := openDB(t)
	saveNoteText(t, db, getFirst(), "n/fresh.md", "chitaiu knigu pro vodu")

	if got := foundBy(t, db, "knige", true); !slices.Equal(got, []string{"n/fresh.md"}) {
		t.Errorf("found %v before any vector exists", got)
	}
	pending, err := db.ChunkQueries().GetUnembeddedChunks(t.Context(), getFirst().ID, testRecipe, chunk.Cursor{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Errorf("the note has %d chunks waiting for a vector, want 1", len(pending))
	}
}

func TestAFragmentIsNotAnAnswerByMeaningButAShortNoteIs(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	vault := getFirst()
	long := "A sentence long enough to say something about the water in the well"
	// A page number in a book is a chunk of its own, and so is a short note.
	if err := db.Chunks().SaveSource(ctx, vault.ID, chunk.Source{Path: "library/book.epub", Kind: "book", Size: 1000, MTime: 1, Hash: "h", Recipe: "epub"}); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().ReplaceChunks(ctx, vault.ID, "book", "library/book.epub", []chunk.Chunk{{
		Start: 0, Length: 200, Text: "12 " + long,
		Small: []chunk.Chunk{{Start: 0, Length: 2, Text: "12"}, {Start: 100, Length: len(long), Text: long}},
	}}); err != nil {
		t.Fatal(err)
	}
	saveNoteText(t, db, vault, "n/short.md", "voda")
	saveSeededVectors(t, db, vault, 0xaa)

	found, err := db.ChunkQueries().FindNearest(ctx, vault.ID, sourceport.NearestQuery{
		Recipe: testRecipe, Vector: getVectorForSeed(0xaa), Limit: 10, Floor: 0.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range found {
		got = append(got, p.Source+"@"+strconv.Itoa(p.HitAt))
	}
	slices.Sort(got)
	if want := []string{"library/book.epub@100", "n/short.md@0"}; !slices.Equal(got, want) {
		t.Errorf("found %v, want %v", got, want)
	}
}
