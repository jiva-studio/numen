package index

import (
	"encoding/hex"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/internal/embedding"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// coarseRows is how many chunks of one source the coarse pass can reach, and
// how many of its small chunks there are to reach.
func coarseRows(t *testing.T, db *DB, vault domain.Vault, path string) (reached, small int) {
	t.Helper()

	target := `FROM chunks c
		JOIN sources s ON s.id = c.source_id
		JOIN vaults v ON v.id = s.vault_id
		WHERE v.identifier = ? AND s.path = ? AND c.parent_id IS NOT NULL`
	small = countRows(t, db, `SELECT COUNT(*) `+target, vault.ID, path)
	reached = countRows(t, db, `SELECT COUNT(*) `+target+
		` AND c.id IN (SELECT chunk_id FROM chunks_vec)`, vault.ID, path)
	if small == 0 {
		t.Fatalf("%s in %s holds no small chunk, so this test would pass either way", path, vault.Name)
	}
	return reached, small
}

// embedFromKeptVectors writes every chunk a vault owes from the vectors the recipe already
// keeps, as the embedding pass does, and says how many it wrote. A chunk whose
// text has no kept vector would cost a model call, and fails the test.
func embedFromKeptVectors(t *testing.T, db *DB, vault domain.Vault) int {
	t.Helper()
	ctx := t.Context()

	owing, err := db.ChunkQueries().GetUnembeddedChunks(ctx, vault.ID, testRecipe, 0, 0, 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	hashes := make([][]byte, 0, len(owing))
	for _, p := range owing {
		hashes = append(hashes, hashOf(t, db, p.Chunk))
	}
	retained, err := db.ChunkQueries().GetKeptVectors(ctx, testRecipe, hashes)
	if err != nil {
		t.Fatal(err)
	}
	vectors := make([]chunk.Vector, 0, len(owing))
	for i, p := range owing {
		value, target := retained[hex.EncodeToString(hashes[i])]
		if !target {
			t.Errorf("chunk %d owes a vector nothing keeps", p.Chunk)
			continue
		}
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Hash: hashes[i], Recipe: testRecipe,
			Value: value, Coarse: embedding.Coarse(embedding.Dimensions(value)),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
		t.Fatal(err)
	}
	return len(vectors)
}

func TestAChunkWhoseTextWasEmbeddedElsewhereIsStillReachedByTheCoarsePass(t *testing.T) {
	// A vector is kept by the text it was made from, so a second chunk holding
	// that text costs no model call. It still needs a coarse row of its own:
	// the coarse pass is the only way a dense search reaches a chunk.
	for _, tc := range []struct {
		name   string
		placed func(t *testing.T, db *DB) (domain.Vault, string)
	}{
		{
			name: "the same book in a second vault",
			placed: func(t *testing.T, db *DB) (domain.Vault, string) {
				book(t, db, first, "library/tale.epub", 0x0f)
				cutBook(t, db, second, "library/tale.epub")
				return second, "library/tale.epub"
			},
		},
		{
			name: "the same book at a second path",
			placed: func(t *testing.T, db *DB) (domain.Vault, string) {
				book(t, db, first, "library/tale.epub", 0x0f)
				cutBook(t, db, first, "copies/tale.epub")
				return first, "copies/tale.epub"
			},
		},
		{
			name: "a second copy whose first is gone",
			placed: func(t *testing.T, db *DB) (domain.Vault, string) {
				book(t, db, first, "library/tale.epub", 0x0f)
				cutBook(t, db, first, "copies/tale.epub")
				if err := db.Chunks().RemoveSources(t.Context(), first.ID, "book", []string{"library/tale.epub"}); err != nil {
					t.Fatal(err)
				}
				return first, "copies/tale.epub"
			},
		},
		{
			name: "a book removed and put back",
			placed: func(t *testing.T, db *DB) (domain.Vault, string) {
				book(t, db, first, "library/tale.epub", 0x0f)
				if err := db.Chunks().RemoveSources(t.Context(), first.ID, "book", []string{"library/tale.epub"}); err != nil {
					t.Fatal(err)
				}
				cutBook(t, db, first, "library/tale.epub")
				return first, "library/tale.epub"
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := openDB(t)
			vault, path := tc.placed(t, db)

			_, small := coarseRows(t, db, vault, path)
			target, isDone, err := db.ChunkQueries().GetProgress(t.Context(), vault.ID, testRecipe)
			if err != nil {
				t.Fatal(err)
			}
			if target-isDone != int64(small) {
				t.Errorf("progress counts %d of %d chunks still to do, want the %d of %s", target-isDone, target, small, path)
			}
			if wrote := embedFromKeptVectors(t, db, vault); wrote != small {
				t.Errorf("%d of the %d small chunks of %s were owed", wrote, small, path)
			}
			if reached, _ := coarseRows(t, db, vault, path); reached != small {
				t.Errorf("the coarse pass reaches %d of the %d small chunks of %s", reached, small, path)
			}
			if again := embedFromKeptVectors(t, db, vault); again != 0 {
				t.Errorf("%d chunks are still owed once every one was written", again)
			}
		})
	}
}

func TestAVectorSavedBackAsItIsCachedPreservesItsRowUntouched(t *testing.T) {
	// A reclaimed vector is written beside its coarse row, and the vector is
	// the one already kept. The guard writes a different vector, so a trigger
	// that never fires cannot pass.
	ctx := t.Context()
	db := openDB(t)
	book(t, db, first, "library/tale.epub", 0x0f)

	for _, statement := range []string{
		`CREATE TABLE rewritten (hash BLOB)`,
		`CREATE TRIGGER count_rewrites AFTER UPDATE ON vectors
		 BEGIN INSERT INTO rewritten VALUES (new.hash); END`,
	} {
		if _, err := db.write.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}

	cutBook(t, db, second, "library/tale.epub")
	if wrote := embedFromKeptVectors(t, db, second); wrote == 0 {
		t.Fatal("nothing was reclaimed")
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM rewritten`); got != 0 {
		t.Errorf("writing back what is kept rewrote %d vectors", got)
	}

	var one int64
	if err := db.read.QueryRowContext(ctx,
		`SELECT id FROM chunks WHERE parent_id IS NOT NULL ORDER BY id LIMIT 1`).Scan(&one); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().SaveVectors(ctx, []chunk.Vector{{
		Chunk: one, Hash: hashOf(t, db, one), Recipe: testRecipe,
		Value: getPreciseBytes(getVectorForSeed(0xf0)), Coarse: bits(0xf0),
	}}); err != nil {
		t.Fatal(err)
	}
	if got := countRows(t, db, `SELECT COUNT(*) FROM rewritten`); got == 0 {
		t.Error("a vector that differs from the one kept was not written, so the count above says nothing")
	}
}
