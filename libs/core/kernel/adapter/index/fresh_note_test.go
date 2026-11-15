package index

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
)

func TestAFreshNoteIsSearchableByItsWordsBeforeItsVector(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	vault := getFirstVault()
	text := "reading a book about water"
	if err := db.Chunks().SaveSource(ctx, vault.ID, chunk.Source{Path: "n/fresh.md", Kind: "note", Size: int64(len(text)), MTime: 1}); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().
		ReplaceChunks(ctx, vault.ID, "note", "n/fresh.md", []chunk.Chunk{{Start: 0, Length: len(text), Text: text, Small: []chunk.Chunk{{Start: 0, Length: len(text), Text: text}}}}); err != nil {
		t.Fatal(err)
	}

	found, err := db.ChunkSearch().FindLexical(ctx, vault.ID, sourcedomain.LexicalQuery{Text: "water", Kinds: nil, Limit: 10, IsGrowing: true})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, p := range found {
		paths = append(paths, p.Path)
	}
	if !slices.Equal(paths, []string{"n/fresh.md"}) {
		t.Errorf("found %v before any vector exists", paths)
	}
	pending, err := db.ChunkVectors().GetUnembeddedChunks(ctx, vault.ID, testRecipe, chunk.Cursor{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Errorf("the note has %d chunks waiting for a vector, want 1", len(pending))
	}
}
