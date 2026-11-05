package index

import (
	"slices"
	"testing"

	notecase "github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// saveCutAt stores notes cut at a chunk size, as a scan that knows the size of the embedding plugin
// hands them over.
func saveCutAt(t *testing.T, db *DB, vault domain.Vault, tokens int, notes ...domain.Note) string {
	t.Helper()
	cut := notecase.NewCutNotes(tokens)
	if err := db.Notes().Save(t.Context(), vault.ID, cut.Execute(notes)); err != nil {
		t.Fatal(err)
	}
	return cut.Recipe()
}

// getNotesCutOtherwise is the paths of the notes of a vault that were not cut by the recipe.
func getNotesCutOtherwise(t *testing.T, db *DB, vault domain.Vault, recipe string) []string {
	t.Helper()
	paths, err := db.ChunkQueries().GetSourcesWithOtherRecipe(t.Context(), vault.ID, string(domain.KindNote), []string{recipe}, 100)
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

func TestANoteIsRecordedWithTheRecipeOfTheCutThatMadeIt(t *testing.T) {
	db := openDB(t)
	recipe := saveCutAt(t, db, getFirst(), 256, noteAt("notes/Hedge.md", "Hedge", wordsOf(80)))

	if recipe == "" {
		t.Fatal("a cut has no recipe")
	}
	if got := getNotesCutOtherwise(t, db, getFirst(), recipe); len(got) != 0 {
		t.Errorf("%v were cut by another recipe than %q, want none", got, recipe)
	}
}

func TestANoteSavedAgainAtAnotherSizeIsRecordedWithTheNewRecipe(t *testing.T) {
	db := openDB(t)
	note := noteAt("notes/Hedge.md", "Hedge", wordsOf(80))
	first := saveCutAt(t, db, getFirst(), 256, note)
	second := saveCutAt(t, db, getFirst(), 512, note)

	if first == second {
		t.Fatalf("cuts at 256 and 512 tokens share the recipe %q", first)
	}
	if got := getNotesCutOtherwise(t, db, getFirst(), second); len(got) != 0 {
		t.Errorf("%v were cut by another recipe than %q after the second save", got, second)
	}
	if got := getNotesCutOtherwise(t, db, getFirst(), first); !slices.Equal(got, []string{"notes/Hedge.md"}) {
		t.Errorf("notes cut by another recipe than %q = %v, want the note saved again", first, got)
	}
}

func TestOnlyTheNotesCutByAnotherRecipeAreReportedAndTheTextOfAFileKeepsItsOwn(t *testing.T) {
	db := openDB(t)
	saveCutAt(t, db, getFirst(), 256, noteAt("notes/Old.md", "Old", wordsOf(80)))
	current := saveCutAt(t, db, getFirst(), 512, noteAt("notes/New.md", "New", wordsOf(80)))
	book(t, db, getFirst(), "library/cut.epub", 0x01)

	if got := getNotesCutOtherwise(t, db, getFirst(), current); !slices.Equal(got, []string{"notes/Old.md"}) {
		t.Errorf("notes cut by another recipe = %v, want only notes/Old.md", got)
	}
	books, err := db.ChunkQueries().GetSourcesWithOtherRecipe(t.Context(), getFirst().ID, "book", []string{"epub"}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 0 {
		t.Errorf("%v were read by another recipe than epub, want the book to keep its own", books)
	}
}

func TestANoteOfAnotherVaultIsNotReportedAsCutByAnotherRecipe(t *testing.T) {
	db := openDB(t)
	recipe := saveCutAt(t, db, getFirst(), 512, noteAt("notes/Hedge.md", "Hedge", wordsOf(80)))
	saveCutAt(t, db, getSecond(), 256, noteAt("notes/Gate.md", "Gate", wordsOf(80)))

	if got := getNotesCutOtherwise(t, db, getFirst(), recipe); len(got) != 0 {
		t.Errorf("the first vault reports %v", got)
	}
}

// A re-cut at another size saves the note again. The chunk whose text is still a chunk keeps its row
// and its vector, and a chunk that is new has none.
func TestACutAtAnotherSizeKeepsTheRowAndVectorOfAChunkWhoseTextDidNotMove(t *testing.T) {
	db := openDB(t)
	path := "notes/Hedge.md"
	kept := domain.Chunk{Start: 0, Length: 12, Text: "gate and shed"}
	moved := domain.Chunk{Start: 13, Length: 19, Text: "compost marrow hedge"}
	whole := func(small ...domain.Chunk) domain.Chunk {
		return domain.Chunk{Start: 0, Length: 40, Text: "Hedge\ngate and shed compost marrow hedge", Small: small}
	}
	note := noteAt(path, "Hedge", "gate and shed compost marrow hedge")
	save := func(recipe string, small ...domain.Chunk) {
		t.Helper()
		if err := db.Notes().Save(t.Context(), getFirst().ID, []domain.NoteChunks{{
			Note: note, Chunks: []domain.Chunk{whole(small...)}, Recipe: recipe,
		}}); err != nil {
			t.Fatal(err)
		}
	}
	rowOf := func(c domain.Chunk) int64 {
		t.Helper()
		rows := rowsOf(t, db, `SELECT c.id FROM chunks c JOIN sources s ON s.id = c.source_id
		                       WHERE s.path = ? AND c.parent_id IS NOT NULL AND c.start = ? AND c.length = ?`,
			path, c.Start, c.Length)
		if len(rows) != 1 {
			t.Fatalf("%d small chunks sit at %d+%d", len(rows), c.Start, c.Length)
		}
		return rows[0]
	}

	save("cut-a", kept, moved)
	saveSeededVectors(t, db, getFirst(), 0x21)
	keptRow := rowOf(kept)
	if !hasVector(t, db, keptRow) {
		t.Fatal("the chunk has no vector before the re-cut")
	}

	fresh := domain.Chunk{Start: 13, Length: 6, Text: "compost"}
	save("cut-b", kept, fresh)

	if got := rowOf(kept); got != keptRow || !hasVector(t, db, got) {
		t.Errorf("the chunk of unchanged text is on row %d with its vector = %v, want row %d with a vector", got, hasVector(t, db, got), keptRow)
	}
	if hasVector(t, db, rowOf(fresh)) {
		t.Error("a chunk cut for the first time has a vector before it was embedded")
	}
}
