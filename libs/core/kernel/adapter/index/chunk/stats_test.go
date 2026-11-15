package chunk_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	indexdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/index"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestGetIndexCounts(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, db *index.DB)
		vault domain.VaultID
		want  indexdomain.Counts
	}{
		{
			name:  "an empty index holds nothing",
			setup: func(*testing.T, *index.DB) {},
			vault: vaultA,
			want:  indexdomain.Counts{},
		},
		{
			name: "a vault the index does not know holds nothing",
			setup: func(t *testing.T, db *index.DB) {
				t.Helper()
				source{path: "a.md", kind: "note", large: "l", smalls: []string{"a"}}.save(t, db)
			},
			vault: nobody,
			want:  indexdomain.Counts{},
		},
		{
			name: "notes only, none embedded",
			setup: func(t *testing.T, db *index.DB) {
				t.Helper()
				source{path: "notes/a.md", kind: "note", large: "l", smalls: []string{"a1", "a2"}}.save(t, db)
				source{path: "notes/b.md", kind: "note", large: "l", smalls: []string{"b1"}}.save(t, db)
			},
			vault: vaultA,
			want: indexdomain.Counts{
				Chunks: 3, Vectors: 0, Sources: 2,
				Extensions: []indexdomain.ExtensionCount{{Extension: "md", Sources: 2, Chunks: 3}},
			},
		},
		{
			name: "pdf chunks are counted by their extension, in the order of the sources",
			setup: func(t *testing.T, db *index.DB) {
				t.Helper()
				source{path: "notes/a.md", kind: "note", large: "l", smalls: []string{"a1"}}.save(t, db)
				source{path: "lib/one.PDF", large: "l", smalls: []string{"p1", "p2", "p3"}}.save(t, db)
				source{path: "lib/two.pdf", large: "l", smalls: []string{"q1"}}.save(t, db)
			},
			vault: vaultA,
			want: indexdomain.Counts{
				Chunks: 5, Vectors: 0, Sources: 3,
				Extensions: []indexdomain.ExtensionCount{
					{Extension: "pdf", Sources: 2, Chunks: 4},
					{Extension: "md", Sources: 1, Chunks: 1},
				},
			},
		},
		{
			name: "embedded chunks count as vectors",
			setup: func(t *testing.T, db *index.DB) {
				t.Helper()
				source{path: "lib/a.epub", large: "l", smalls: []string{"a1", "a2", "a3"}}.save(t, db)
				source{path: "lib/b.pdf", large: "l", smalls: []string{"b1"}}.save(t, db)
				embed(t, db, vaultA, "lib/a.epub", 0xF0)
			},
			vault: vaultA,
			want: indexdomain.Counts{
				Chunks: 4, Vectors: 3, Sources: 2,
				Extensions: []indexdomain.ExtensionCount{
					{Extension: "epub", Sources: 1, Chunks: 3},
					{Extension: "pdf", Sources: 1, Chunks: 1},
				},
			},
		},
		{
			name: "a deleted source leaves no chunk and no vector",
			setup: func(t *testing.T, db *index.DB) {
				t.Helper()
				source{path: "lib/a.epub", large: "l", smalls: []string{"a1", "a2"}}.save(t, db)
				source{path: "lib/b.epub", large: "l", smalls: []string{"b1"}}.save(t, db)
				embed(t, db, vaultA, "lib/a.epub", 0xF0)
				embed(t, db, vaultA, "lib/b.epub", 0xF0)
				if err := db.Chunks().RemoveSources(t.Context(), vaultA, "book", []string{"lib/a.epub"}); err != nil {
					t.Fatal(err)
				}
			},
			vault: vaultA,
			want: indexdomain.Counts{
				Chunks: 1, Vectors: 1, Sources: 1,
				Extensions: []indexdomain.ExtensionCount{{Extension: "epub", Sources: 1, Chunks: 1}},
			},
		},
		{
			name: "a path without an extension, or with a dot in a folder only, has none",
			setup: func(t *testing.T, db *index.DB) {
				t.Helper()
				source{path: "README", large: "l", smalls: []string{"r"}}.save(t, db)
				source{path: "v1.2/notes", large: "l", smalls: []string{"n"}}.save(t, db)
				source{path: "archive.tar.gz", large: "l", smalls: []string{"z"}}.save(t, db)
			},
			vault: vaultA,
			want: indexdomain.Counts{
				Chunks: 3, Vectors: 0, Sources: 3,
				Extensions: []indexdomain.ExtensionCount{
					{Extension: "", Sources: 2, Chunks: 2},
					{Extension: "gz", Sources: 1, Chunks: 1},
				},
			},
		},
		{
			name: "another vault is not counted",
			setup: func(t *testing.T, db *index.DB) {
				t.Helper()
				source{path: "a.md", kind: "note", large: "l", smalls: []string{"a"}}.save(t, db)
				source{vault: vaultB, path: "b.md", kind: "note", large: "l", smalls: []string{"b1", "b2"}}.save(t, db)
			},
			vault: vaultA,
			want: indexdomain.Counts{
				Chunks: 1, Vectors: 0, Sources: 1,
				Extensions: []indexdomain.ExtensionCount{{Extension: "md", Sources: 1, Chunks: 1}},
			},
		},
	}
	for _, one := range tests {
		t.Run(one.name, func(t *testing.T) {
			db := openIndex(t)
			one.setup(t, db)

			got, err := db.ChunkVectors().GetIndexCounts(t.Context(), one.vault, recipe)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, one.want) {
				t.Errorf("got %+v, want %+v", got, one.want)
			}
		})
	}
}

func TestGetIndexCountsFailsOnABrokenIndex(t *testing.T) {
	if _, err := closed(t).GetIndexCounts(t.Context(), vaultA, recipe); err == nil {
		t.Error("a closed index answered")
	}
	removed := broken(t, func(*index.DB) {}, `ALTER TABLE chunks RENAME TO gone`)
	if _, err := removed.GetIndexCounts(t.Context(), vaultA, recipe); err == nil {
		t.Error("an index with no chunks answered")
	}
}

func TestGetIndexSizes(t *testing.T) {
	db := openIndex(t)
	empty, err := db.ChunkVectors().GetIndexSizes(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if empty.Database <= 0 {
		t.Errorf("an empty index takes %d bytes, want some", empty.Database)
	}

	source{path: "lib/a.epub", large: "l", smalls: []string{"a1", "a2", "a3"}}.save(t, db)
	embed(t, db, vaultA, "lib/a.epub", 0xF0)
	full, err := db.ChunkVectors().GetIndexSizes(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if full.Vectors <= empty.Vectors {
		t.Errorf("vectors take %d bytes after saving three, and %d before", full.Vectors, empty.Vectors)
	}
	if full.Database < full.Vectors {
		t.Errorf("the database takes %d bytes, fewer than its vectors, %d", full.Database, full.Vectors)
	}
}

func TestGetIndexSizesFailsOnAClosedIndex(t *testing.T) {
	if _, err := closed(t).GetIndexSizes(t.Context()); err == nil {
		t.Error("a closed index answered")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := openIndex(t).ChunkVectors().GetIndexSizes(ctx); err == nil {
		t.Error("a canceled call answered")
	}
}

// A chunk shorter than the minimum length carries no vector, so it is left out of both counts, the
// total and the one by extension. The smallest chunks of the fixture hold 50 bytes.
func TestGetIndexCountsBelowTheMinimum(t *testing.T) {
	tests := []struct {
		name    string
		minimum int
		want    indexdomain.Counts
	}{
		{
			name: "at the length of the chunks all are counted", minimum: 50,
			want: indexdomain.Counts{
				Chunks: 3, Vectors: 3, Sources: 1,
				Extensions: []indexdomain.ExtensionCount{{Extension: "epub", Sources: 1, Chunks: 3}},
			},
		},
		{
			name: "above the length of the chunks none is counted", minimum: 51,
			want: indexdomain.Counts{
				Chunks: 0, Vectors: 0, Sources: 1,
				Extensions: []indexdomain.ExtensionCount{{Extension: "epub", Sources: 1, Chunks: 0}},
			},
		},
	}
	for _, one := range tests {
		t.Run(one.name, func(t *testing.T) {
			db := openIndex(t)
			source{path: "lib/a.epub", large: "l", smalls: []string{"a1", "a2", "a3"}}.save(t, db)
			embed(t, db, vaultA, "lib/a.epub", 0xF0)
			if _, err := db.VectorIndex().ApplyMinChunkLength(t.Context(), one.minimum); err != nil {
				t.Fatal(err)
			}

			got, err := db.ChunkVectors().GetIndexCounts(t.Context(), vaultA, recipe)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, one.want) {
				t.Errorf("got %+v, want %+v", got, one.want)
			}
		})
	}
}
