package index

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func saveChunkedAt(t *testing.T, db *DB, vault domain.Vault, tokens int, notes ...domain.Note) string {
	t.Helper()
	splitter := chunking.NewCutNotes(tokens)
	if err := db.Notes().Save(t.Context(), vault.ID, splitter.Execute(notes)); err != nil {
		t.Fatal(err)
	}
	return splitter.Recipe()
}

func getNotesWithOtherRecipe(t *testing.T, db *DB, vault domain.Vault, recipe string) []string {
	t.Helper()
	paths, err := db.ChunkFiles().GetSourcesWithOtherRecipe(t.Context(), vault.ID, string(domain.KindNote), []string{recipe}, 100)
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

func TestANoteIsRecordedWithTheChunkingRecipe(t *testing.T) {
	db := openDB(t)
	recipe := saveChunkedAt(t, db, getFirstVault(), 256, noteAt("notes/Hedge.md", "Hedge", wordsOf(80)))

	if recipe == "" {
		t.Fatal("a cut has no recipe")
	}
	if got := getNotesWithOtherRecipe(t, db, getFirstVault(), recipe); len(got) != 0 {
		t.Errorf("%v were cut by another recipe than %q, want none", got, recipe)
	}
}

func TestANoteSavedAgainAtAnotherSizeIsRecordedWithTheNewRecipe(t *testing.T) {
	db := openDB(t)
	note := noteAt("notes/Hedge.md", "Hedge", wordsOf(80))
	first := saveChunkedAt(t, db, getFirstVault(), 256, note)
	second := saveChunkedAt(t, db, getFirstVault(), 512, note)

	if first == second {
		t.Fatalf("cuts at 256 and 512 tokens share the recipe %q", first)
	}
	if got := getNotesWithOtherRecipe(t, db, getFirstVault(), second); len(got) != 0 {
		t.Errorf("%v were cut by another recipe than %q after the second save", got, second)
	}
	if got := getNotesWithOtherRecipe(t, db, getFirstVault(), first); !slices.Equal(got, []string{"notes/Hedge.md"}) {
		t.Errorf("notes cut by another recipe than %q = %v, want the note saved again", first, got)
	}
}

func TestOnlyNotesWithAnotherChunkingRecipeAreReportedAndAFileTextKeepsItsOwn(t *testing.T) {
	db := openDB(t)
	saveChunkedAt(t, db, getFirstVault(), 256, noteAt("notes/Old.md", "Old", wordsOf(80)))
	current := saveChunkedAt(t, db, getFirstVault(), 512, noteAt("notes/New.md", "New", wordsOf(80)))
	book(t, db, getFirstVault(), "library/cut.epub", 0x01)

	if got := getNotesWithOtherRecipe(t, db, getFirstVault(), current); !slices.Equal(got, []string{"notes/Old.md"}) {
		t.Errorf("notes cut by another recipe = %v, want only notes/Old.md", got)
	}
	books, err := db.ChunkFiles().GetSourcesWithOtherRecipe(t.Context(), getFirstVault().ID, "book", []string{"epub"}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 0 {
		t.Errorf("%v were read by another recipe than epub, want the book to keep its own", books)
	}
}

func TestANoteOfAnotherVaultIsNotReportedForItsChunkingRecipe(t *testing.T) {
	db := openDB(t)
	recipe := saveChunkedAt(t, db, getFirstVault(), 512, noteAt("notes/Hedge.md", "Hedge", wordsOf(80)))
	saveChunkedAt(t, db, getSecond(), 256, noteAt("notes/Gate.md", "Gate", wordsOf(80)))

	if got := getNotesWithOtherRecipe(t, db, getFirstVault(), recipe); len(got) != 0 {
		t.Errorf("the first vault reports %v", got)
	}
}

// A re-cut at another size saves the note again. The chunk whose text is still a chunk keeps its row
// and its vector, and a chunk that is new has none.
func TestAnotherChunkSizeKeepsTheRowAndVectorOfAChunkWhoseTextIsUnchanged(t *testing.T) {
	db := openDB(t)
	path := "notes/Hedge.md"
	unchanged := domain.Chunk{Start: 0, Length: 12, Text: "gate and shed"}
	moved := domain.Chunk{Start: 13, Length: 19, Text: "compost marrow hedge"}
	whole := func(small ...domain.Chunk) domain.Chunk {
		return domain.Chunk{Start: 0, Length: 40, Text: "Hedge\ngate and shed compost marrow hedge", Small: small}
	}
	note := noteAt(path, "Hedge", "gate and shed compost marrow hedge")
	save := func(recipe string, small ...domain.Chunk) {
		t.Helper()
		if err := db.Notes().Save(t.Context(), getFirstVault().ID, []domain.NoteChunks{{
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

	save("cut-a", unchanged, moved)
	saveSeededVectors(t, db, getFirstVault(), 0x21)
	unchangedRow := rowOf(unchanged)
	if !hasVector(t, db, unchangedRow) {
		t.Fatal("the chunk has no vector before the re-cut")
	}

	added := domain.Chunk{Start: 13, Length: 6, Text: "compost"}
	save("cut-b", unchanged, added)

	if got := rowOf(unchanged); got != unchangedRow || !hasVector(t, db, got) {
		t.Errorf("the chunk of unchanged text is on row %d with its vector = %v, want row %d with a vector", got, hasVector(t, db, got), unchangedRow)
	}
	if hasVector(t, db, rowOf(added)) {
		t.Error("a chunk cut for the first time has a vector before it was embedded")
	}
}
