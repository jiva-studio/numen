package chunk_test

import (
	"errors"
	"slices"
	"testing"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func TestListDocumentsFindsAComposedPathByADecomposedPrefixWithItsTitleAndVectorCount(t *testing.T) {
	composed := norm.NFC.String("Шастры/йога.md")
	queries := broken(t, func(db *index.DB) {
		source{path: composed, kind: "note", large: "large text", smalls: []string{"first small", "second small"}}.save(t, db)
		source{path: "other/b.md", kind: "note", large: "other text", smalls: []string{"third small"}}.save(t, db)
		embed(t, db, vaultA, composed, 0xF0)
	}, "INSERT INTO notes (source_id, vault_id, folded_name, title) SELECT id, vault_id, 'йога', 'Йога' FROM sources WHERE path = '"+composed+"'")

	got, err := queries.Search.ListDocuments(t.Context(), vaultA, searchquery.DocumentQuery{
		PathPrefixes: []string{norm.NFD.String("Шастры/")}, Recipe: recipe,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != composed || got[0].Title != "Йога" || got[0].SourceKind != domain.KindNote {
		t.Fatalf("listed %+v, want the composed note with its title", got)
	}
	if got[0].ChunkCount != 2 {
		t.Errorf("the note counts %d small chunks with a vector, want 2", got[0].ChunkCount)
	}
	other, err := queries.Search.ListDocuments(t.Context(), vaultA, searchquery.DocumentQuery{PathPrefixes: []string{"other/"}, Recipe: recipe})
	if err != nil || len(other) != 1 || other[0].ChunkCount != 0 {
		t.Errorf("a note without vectors listed %+v, %v, want a count of 0", other, err)
	}
	all, err := queries.Search.ListDocuments(t.Context(), vaultA, searchquery.DocumentQuery{Recipe: recipe, Limit: 1})
	if err != nil || len(all) != 2 {
		t.Errorf("a limit of 1 listed %d documents, %v, want 2 (limit + 1)", len(all), err)
	}
}

func TestListDocumentVectorsPagesByChunkIDWithinThePaths(t *testing.T) {
	db := openIndex(t)
	source{path: "a.md", kind: "note", large: "a", smalls: []string{"a one", "a two", "a three"}}.save(t, db)
	source{path: "b.md", kind: "note", large: "b", smalls: []string{"b one"}}.save(t, db)
	embed(t, db, vaultA, "a.md", 0xF0)
	embed(t, db, vaultA, "b.md", 0x0F)
	search := db.ChunkSearch()

	first, err := search.ListDocumentVectors(t.Context(), vaultA, searchquery.DocumentVectorQuery{Paths: []string{"a.md"}, Recipe: recipe, Limit: 2})
	if err != nil || len(first) != 2 {
		t.Fatalf("the first page is %d vectors, %v", len(first), err)
	}
	rest, err := search.ListDocumentVectors(t.Context(), vaultA, searchquery.DocumentVectorQuery{Paths: []string{"a.md"}, Recipe: recipe, Limit: 2, After: first[1].Chunk})
	if err != nil || len(rest) != 1 {
		t.Fatalf("the second page is %d vectors, %v", len(rest), err)
	}
	ids := []domain.ChunkID{first[0].Chunk, first[1].Chunk, rest[0].Chunk}
	if slices.Contains(ids[:2], rest[0].Chunk) {
		t.Errorf("the pages repeat a chunk: %v", ids)
	}
}

func TestFindWordsKeepsToThePathsBestFirst(t *testing.T) {
	db := openIndex(t)
	source{path: "a.md", kind: "note", large: "a", smalls: []string{"tide", "tide tide tide and the tide"}}.save(t, db)
	source{path: "b.md", kind: "note", large: "b", smalls: []string{"tide tide"}}.save(t, db)
	search := db.ChunkSearch()

	got, err := search.FindWords(t.Context(), vaultA, searchquery.WordQuery{Expression: "tide", Paths: []string{"a.md"}, Limit: 10})
	if err != nil || len(got) != 2 {
		t.Fatalf("found %+v, %v, want the two chunks of a.md", got, err)
	}
	if got[0].Score < got[1].Score {
		t.Errorf("the scores %v, %v are not best first", got[0].Score, got[1].Score)
	}
	everywhere, err := search.FindWords(t.Context(), vaultA, searchquery.WordQuery{Expression: "tide", Limit: 10})
	if err != nil || len(everywhere) != 3 {
		t.Errorf("found %d chunks in the vault, %v, want 3", len(everywhere), err)
	}
	_, err = search.FindWords(t.Context(), vaultA, searchquery.WordQuery{Expression: `"unterminated`, Limit: 10})
	if !errors.Is(err, sourcedomain.ErrInvalidWords) {
		t.Errorf("an invalid expression returned %v", err)
	}
}

func TestGetWordScoresOmitsTheChunksThatDoNotMatch(t *testing.T) {
	db := openIndex(t)
	source{path: "a.md", kind: "note", large: "a", smalls: []string{"tide", "stone"}}.save(t, db)
	search := db.ChunkSearch()
	all, err := search.FindWords(t.Context(), vaultA, searchquery.WordQuery{Expression: "tide OR stone", Limit: 10})
	if err != nil || len(all) != 2 {
		t.Fatalf("found %+v, %v", all, err)
	}
	scores, err := search.GetWordScores(t.Context(), vaultA, "tide", []domain.ChunkID{all[0].Chunk, all[1].Chunk})
	if err != nil || len(scores) != 1 {
		t.Errorf("scores %v, %v, want one chunk", scores, err)
	}
}

func TestListDocumentsWithinPathsReturnsOnlyTheSourcesAtThem(t *testing.T) {
	db := openIndex(t)
	source{path: "a.md", kind: "note", large: "a", smalls: []string{"a one"}}.save(t, db)
	source{path: "a.md.md", kind: "note", large: "b", smalls: []string{"b one"}}.save(t, db)
	source{path: "c.md", kind: "note", large: "c", smalls: []string{"c one"}}.save(t, db)
	search := db.ChunkSearch()

	got, err := search.ListDocuments(t.Context(), vaultA, searchquery.DocumentQuery{
		Paths: []string{"a.md", "c.md", "gone.md"}, IsWithinPaths: true, Recipe: recipe,
	})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, d := range got {
		paths = append(paths, d.Path)
	}
	if !slices.Equal(paths, []string{"a.md", "c.md"}) {
		t.Errorf("listed %v, want a.md and c.md", paths)
	}
	none, err := search.ListDocuments(t.Context(), vaultA, searchquery.DocumentQuery{IsWithinPaths: true, Recipe: recipe})
	if err != nil || len(none) != 0 {
		t.Errorf("no paths listed %d documents, %v, want none", len(none), err)
	}
}
