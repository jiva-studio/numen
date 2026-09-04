package index

import (
	"encoding/hex"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func TestEveryConnectionGetsThePragmas(t *testing.T) {
	// The bug this guards against is invisible to an ordinary test: executing
	// `PRAGMA foreign_keys = ON` after opening configures one connection of the
	// pool, and a sequential test keeps being handed that same connection. Only
	// holding several at once shows the rest were never configured.
	ctx := t.Context()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const conns = 8
	db.read.SetMaxOpenConns(conns)

	var wg sync.WaitGroup
	foreignKeys := make([]int, conns)
	busyTimeout := make([]int, conns)
	synchronous := make([]int, conns)
	release := make(chan struct{})

	for i := range conns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := db.read.Conn(ctx)
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()
			if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys[i]); err != nil {
				t.Error(err)
				return
			}
			if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout[i]); err != nil {
				t.Error(err)
				return
			}
			if err := conn.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&synchronous[i]); err != nil {
				t.Error(err)
				return
			}
			// Hold the connection so the next goroutine is forced to open a new
			// one instead of reusing this one.
			<-release
		}()
	}
	// Give every goroutine time to take a distinct connection.
	for range conns {
		release <- struct{}{}
	}
	wg.Wait()

	for i := range conns {
		if foreignKeys[i] != 1 {
			t.Errorf("connection %d has foreign_keys = %d, want 1", i, foreignKeys[i])
		}
		if busyTimeout[i] != 5000 {
			t.Errorf("connection %d has busy_timeout = %d, want 5000", i, busyTimeout[i])
		}
		// NORMAL is 1. FULL asks the disk to settle at every commit, which a
		// rebuild does thousands of times, and nothing else in the suite
		// notices which one is set.
		if synchronous[i] != 1 {
			t.Errorf("connection %d has synchronous = %d, want 1 (NORMAL)", i, synchronous[i])
		}
	}
}

func TestDSNCarriesEveryPragma(t *testing.T) {
	got := dsn("/tmp/index.db")
	for _, want := range []string{"foreign_keys%281%29", "busy_timeout%285000%29", "journal_mode%28WAL%29", "synchronous%28NORMAL%29"} {
		if !strings.Contains(got, want) {
			t.Errorf("dsn %q is missing %s", got, want)
		}
	}
	if !strings.HasPrefix(got, "/tmp/index.db?") {
		t.Errorf("dsn %q does not start with the path", got)
	}
}

func TestVectorSearchIsInTheBuild(t *testing.T) {
	// The extension is bundled with the driver, so a build without it fails at
	// the first query. The
	// version floor in go.mod does not say the functions are there; this does.
	db, err := Open(t.Context(), filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var version string
	if err := db.read.QueryRowContext(t.Context(), "SELECT vec_version()").Scan(&version); err != nil {
		t.Fatalf("vec_version() is not available: %v", err)
	}
	if version == "" {
		t.Error("vec_version() returned nothing")
	}
}

// A vector index is built for one width, and a model of another width cannot be
// written to it. The width is the model's, so an installation that names a
// 384-dimension model can embed with it.
func TestTheVectorIndexIsFittedToTheModel(t *testing.T) {
	ctx := t.Context()
	db := opened(t)

	if err := db.FitVectors(ctx, 384, narrowRecipe(384)); err != nil {
		t.Fatal(err)
	}
	if err := db.FitVectors(ctx, 384, narrowRecipe(384)); err != nil {
		t.Fatalf("fitting to the width it already holds: %v", err)
	}

	narrow(t, db, first, 384)
}

// Fitting the coarse index to another width leaves what was made standing, and
// a width that comes back is filled from it.
//
// A vector of the old width is not comparable with a query of the new one, and
// is kept by the text it was bought for.
func TestFittingToAnotherWidthKeepsWhatWasMade(t *testing.T) {
	ctx := t.Context()
	db := opened(t)

	if err := db.FitVectors(ctx, 384, narrowRecipe(384)); err != nil {
		t.Fatal(err)
	}
	narrow(t, db, first, 384)
	made := counted(t, db, `SELECT count(*) FROM vectors`)
	if made == 0 {
		t.Fatal("nothing was kept for the text that was embedded")
	}
	if coarse := counted(t, db, `SELECT count(*) FROM chunks_vec`); coarse != made {
		t.Fatalf("%d of %d vectors reached the coarse index", coarse, made)
	}

	if err := db.FitVectors(ctx, 1024, narrowRecipe(1024)); err != nil {
		t.Fatal(err)
	}
	if left := counted(t, db, `SELECT count(*) FROM vectors`); left != made {
		t.Errorf("%d of %d vectors survived a change of width", left, made)
	}

	if err := db.FitVectors(ctx, 384, narrowRecipe(384)); err != nil {
		t.Fatal(err)
	}
	if back := counted(t, db, `SELECT count(*) FROM chunks_vec`); back != made {
		t.Errorf("the width came back and %d of %d vectors are in the coarse index", back, made)
	}
}

// narrowRecipe is what a model of the width given writes its vectors under.
func narrowRecipe(dims int) string { return "model/" + strconv.Itoa(dims) }

// narrow cuts one source and embeds its small chunks at the width given.
func narrow(t *testing.T, db *DB, vault domain.Vault, dims int) {
	t.Helper()
	ctx := t.Context()

	path := "library/" + strconv.Itoa(dims) + ".epub"
	if err := db.Chunks().SaveSource(ctx, vault.ID, chunk.Source{
		Path: path, Kind: "book", Size: 100, MTime: 1, Hash: "h" + path, Recipe: "epub",
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Chunks().SaveChunks(ctx, vault.ID, "book", path, []chunk.Chunk{{
		Start: 0, Length: 50, Text: "whole",
		Small: []chunk.Chunk{{Start: 0, Length: 50, Text: "a chunk of text"}},
	}}); err != nil {
		t.Fatal(err)
	}

	owing, err := db.ChunkQueries().Unembedded(ctx, vault.ID, narrowRecipe(dims), 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) == 0 {
		t.Fatal("nothing owes a vector, so this proves nothing")
	}
	vectors := make([]chunk.Vector, 0, len(owing))
	for _, p := range owing {
		// A vector is found by the address of the text the chunk holds, which
		// is the chunk's own hash.
		hash, err := hex.DecodeString(p.ChunkHash)
		if err != nil || len(hash) == 0 {
			t.Fatalf("chunk %d owes a vector under hash %q", p.Chunk, p.ChunkHash)
		}
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Hash: hash, Recipe: narrowRecipe(dims),
			Value: make([]byte, dims), Coarse: make([]byte, dims/8),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
		t.Fatalf("a %d-dimension vector could not be written: %v", dims, err)
	}
}
