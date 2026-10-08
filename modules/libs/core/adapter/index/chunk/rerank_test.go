package chunk

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
	_ "modernc.org/sqlite"
)

func setupTestDB(t testing.TB, count int, dims int) (*Queries, []int64, []float32) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rerank_test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
	CREATE TABLE chunks (
		id INTEGER PRIMARY KEY,
		vault_id INTEGER,
		source_id INTEGER,
		start INTEGER,
		length INTEGER,
		location TEXT,
		text TEXT,
		hash TEXT,
		parent INTEGER
	);
	CREATE TABLE sources (
		id INTEGER PRIMARY KEY,
		vault_id INTEGER,
		path TEXT,
		kind TEXT,
		size INTEGER,
		mtime INTEGER,
		hash TEXT,
		recipe TEXT,
		producer TEXT
	);
	CREATE TABLE vectors (
		hash BLOB,
		recipe TEXT,
		embedding BLOB,
		PRIMARY KEY (hash, recipe)
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}

	query := make([]float32, dims)
	for i := range query {
		query[i] = 1.0
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	stmtSource, err := tx.Prepare(`INSERT INTO sources (id, vault_id, path, kind, size, mtime, hash, recipe, producer) VALUES (?, 1, ?, ?, 1000, 1, ?, 'epub', '')`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmtSource.Close()

	stmtChunk, err := tx.Prepare(`INSERT INTO chunks (id, vault_id, source_id, start, length, location, text, hash, parent) VALUES (?, 1, ?, ?, ?, 'loc', 'text', ?, 0)`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmtChunk.Close()

	stmtVector, err := tx.Prepare(`INSERT INTO vectors (hash, recipe, embedding) VALUES (?, 'recipe1', ?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmtVector.Close()

	candidates := make([]int64, 0, count)
	for i := 1; i <= count; i++ {
		sourceKind := "book"
		if i%3 == 0 {
			sourceKind = "note"
		}
		sourceHash := fmt.Sprintf("src-%04d", i)
		if _, err := stmtSource.Exec(i, fmt.Sprintf("path/%d", i), sourceKind, sourceHash); err != nil {
			t.Fatal(err)
		}

		chunkHashBytes := sha256.Sum256([]byte(fmt.Sprintf("chunk-%d", i)))
		chunkHashHex := hex.EncodeToString(chunkHashBytes[:])

		if _, err := stmtChunk.Exec(i, i, 0, 100, chunkHashHex); err != nil {
			t.Fatal(err)
		}

		vec := make([]float32, dims)
		for d := range vec {
			if (d+i)%2 == 0 {
				vec[d] = 1.0
			} else {
				vec[d] = -1.0
			}
		}
		norm := embedding.Normalise(vec)
		qBytes := embedding.Bytes(norm)
		stored := make([]byte, len(qBytes))
		for d, v := range qBytes {
			stored[d] = byte(v)
		}

		if _, err := stmtVector.Exec(chunkHashBytes[:], stored); err != nil {
			t.Fatal(err)
		}
		candidates = append(candidates, int64(i))
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	return NewQueries(db), candidates, query
}

func TestRerank(t *testing.T) {
	ctx := t.Context()
	queries, candidates, query := setupTestDB(t, 20, 1024)

	t.Run("basic ranking and floor", func(t *testing.T) {
		ranked, err := queries.rerank(ctx, "recipe1", query, candidates, nil, -1.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ranked) != len(candidates) {
			t.Fatalf("expected %d candidates, got %d", len(candidates), len(ranked))
		}

		filtered, err := queries.rerank(ctx, "recipe1", query, candidates, nil, 0.5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(filtered) >= len(ranked) {
			t.Fatalf("expected floor to filter candidates, got %d vs %d", len(filtered), len(ranked))
		}
	})

	t.Run("empty candidates", func(t *testing.T) {
		ranked, err := queries.rerank(ctx, "recipe1", query, nil, nil, 0.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ranked) != 0 {
			t.Fatalf("expected 0 candidates, got %d", len(ranked))
		}
	})

	t.Run("kind filtering", func(t *testing.T) {
		notesOnly, err := queries.rerank(ctx, "recipe1", query, candidates, []domain.SourceKind{"note"}, -1.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notesOnly) != 6 {
			t.Fatalf("expected 6 note chunks, got %d", len(notesOnly))
		}
		for _, id := range notesOnly {
			if id%3 != 0 {
				t.Fatalf("expected chunk id to be multiple of 3, got %d", id)
			}
		}
	})

	t.Run("preserves coarse order on tie", func(t *testing.T) {
		subCandidates := []int64{1, 3, 5}
		ranked, err := queries.rerank(ctx, "recipe1", query, subCandidates, nil, -1.0)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(ranked, subCandidates) {
			t.Fatalf("expected order %v, got %v", subCandidates, ranked)
		}
	})
}

func BenchmarkRerank(b *testing.B) {
	ctx := b.Context()
	for _, size := range []int{10, 50, 100, 200} {
		b.Run(fmt.Sprintf("Candidates_%d", size), func(b *testing.B) {
			queries, candidates, query := setupTestDB(b, size, 1024)
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				out, err := queries.rerank(ctx, "recipe1", query, candidates, nil, -1.0)
				if err != nil {
					b.Fatal(err)
				}
				if len(out) != size {
					b.Fatalf("expected %d, got %d", size, len(out))
				}
			}
		})
	}
}
