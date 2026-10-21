package index

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

const otherRecipe = "other-model"

// saveSeededVectorsUnder gives every chunk of a vault that owes a vector under the
// recipe one at the seed given.
func saveSeededVectorsUnder(t *testing.T, db *DB, vault domain.Vault, recipe string, seed byte) int {
	t.Helper()
	ctx := t.Context()

	owing, err := db.ChunkQueries().GetUnembeddedChunks(ctx, vault.ID, recipe, 0, 0, 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	vectors := make([]chunk.Vector, 0, len(owing))
	for _, p := range owing {
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Hash: hashOf(t, db, p.Chunk), Recipe: recipe,
			Value: getPreciseBytes(getVectorForSeed(seed)), Coarse: bits(seed),
		})
	}
	if err := db.Chunks().SaveVectors(ctx, vectors); err != nil {
		t.Fatal(err)
	}
	return len(vectors)
}

// storedBits is the coarse form every chunk holds in the coarse index.
func storedBits(t *testing.T, db *DB) map[int64][]byte {
	t.Helper()
	rows, err := db.read.QueryContext(t.Context(), `SELECT chunk_id, coarse FROM chunks_vec`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	target := map[int64][]byte{}
	for rows.Next() {
		var id int64
		var value []byte
		if err := rows.Scan(&id, &value); err != nil {
			t.Fatal(err)
		}
		target[id] = value
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return target
}

func fit(t *testing.T, db *DB, recipe string) {
	t.Helper()
	if err := db.FitVectors(t.Context(), 1024, recipe); err != nil {
		t.Fatal(err)
	}
}

func TestSwitchingBetweenTwoModelsOfOneWidthAndBackRestoresTheFirstModelsBits(t *testing.T) {
	db := openDB(t)
	cutBook(t, db, first, "library/tale.epub")
	fit(t, db, testRecipe)
	small := saveSeededVectorsUnder(t, db, first, testRecipe, 0x0f)

	fit(t, db, otherRecipe)
	if got := saveSeededVectorsUnder(t, db, first, otherRecipe, 0xf0); got != small {
		t.Fatalf("%d of %d chunks were owed under the other recipe", got, small)
	}
	for id, got := range storedBits(t, db) {
		if !bytes.Equal(got, bits(0xf0)) {
			t.Fatalf("chunk %d holds %x while the other recipe fills the index", id, got[:2])
		}
	}

	fit(t, db, testRecipe)
	target := storedBits(t, db)
	if len(target) != small {
		t.Fatalf("%d coarse rows, want %d", len(target), small)
	}
	for id, got := range target {
		if !bytes.Equal(got, bits(0x0f)) {
			t.Errorf("chunk %d holds %x, want the bits of the recipe in use", id, got[:2])
		}
	}
	if owed := saveSeededVectorsUnder(t, db, first, testRecipe, 0xaa); owed != 0 {
		t.Errorf("%d chunks are owed under a recipe that holds every vector", owed)
	}
}

func TestFittingTheRecipeAndWidthAlreadyHeldPreservesTheCoarseIndexAlone(t *testing.T) {
	db := openDB(t)
	book(t, db, first, "library/tale.epub", 0x0f)
	fit(t, db, testRecipe)

	// A row no chunk owns survives only if the index is not rebuilt.
	if _, err := db.write.ExecContext(t.Context(),
		`INSERT INTO chunks_vec (chunk_id, vault_id, coarse) VALUES (999999, 1, vec_bit(?))`, bits(0x55)); err != nil {
		t.Fatal(err)
	}
	fit(t, db, testRecipe)
	if _, target := storedBits(t, db)[999999]; !target {
		t.Error("fitting the recipe already held rebuilt the coarse index")
	}

	fit(t, db, otherRecipe)
	if _, targetVal := storedBits(t, db)[999999]; targetVal {
		t.Error("fitting another recipe left a row of the old one")
	}
}

func TestAnIndexThatRecordsNoRecipeIsFilledOnceAndThenRecorded(t *testing.T) {
	db := openDB(t)
	book(t, db, first, "library/tale.epub", 0x0f)
	fit(t, db, testRecipe)

	// An index from before the recipe was recorded, holding another model's bits.
	if _, err := db.write.ExecContext(t.Context(), `DELETE FROM coarse_recipe`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.write.ExecContext(t.Context(), `DELETE FROM chunks_vec`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.write.ExecContext(t.Context(),
		`INSERT INTO chunks_vec (chunk_id, vault_id, coarse) SELECT id, vault_id, vec_bit(?) FROM chunks WHERE parent_id IS NOT NULL`,
		bits(0xf0)); err != nil {
		t.Fatal(err)
	}

	fit(t, db, testRecipe)
	target := storedBits(t, db)
	if len(target) == 0 {
		t.Fatal("the index was not filled")
	}
	for id, got := range target {
		if !bytes.Equal(got, bits(0x0f)) {
			t.Errorf("chunk %d still holds %x", id, got[:2])
		}
	}
	if got, recorded, err := coarseRecipe(t.Context(), db.write); err != nil || !recorded || got != testRecipe {
		t.Errorf("recorded recipe = %q, %v, %v", got, recorded, err)
	}
}

func TestAChunkTheNewRecipeContainsNoVectorForGetsNoCoarseRow(t *testing.T) {
	db := openDB(t)
	book(t, db, first, "library/tale.epub", 0x0f)
	fit(t, db, testRecipe)

	fit(t, db, otherRecipe)
	if rows := len(storedBits(t, db)); rows != 0 {
		t.Errorf("%d coarse rows hold another recipe's bits", rows)
	}
	owing, err := db.ChunkQueries().GetUnembeddedChunks(t.Context(), first.ID, otherRecipe, 0, 0, 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) == 0 {
		t.Error("no chunk is owed under a recipe that holds no vector")
	}
}

func TestMigrationFourAppliesOnAnIndexAtTheVersionBeforeIt(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`DROP TABLE coarse_recipe`,
		`DELETE FROM schema_migrations WHERE version = 4`,
		`PRAGMA user_version = 3`,
	} {
		if _, err := db.write.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	again, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	if got := getNewestVersion(t); got < 4 {
		t.Fatalf("newest version = %d", got)
	}
	if _, recorded, err := coarseRecipe(ctx, again.write); err != nil || recorded {
		t.Fatalf("a migrated index records %v, %v before anything filled it", recorded, err)
	}
	if err := again.FitVectors(ctx, 1024, testRecipe); err != nil {
		t.Fatal(err)
	}
	if got, recorded, err := coarseRecipe(ctx, again.write); err != nil || !recorded || got != testRecipe {
		t.Errorf("recorded recipe = %q, %v, %v", got, recorded, err)
	}
	if _, err := again.write.ExecContext(ctx, `INSERT INTO coarse_recipe (id, recipe) VALUES (2, 'x')`); err == nil {
		t.Error("a second row was accepted")
	}
}

func TestAFreshIndexContainsTheRecipeTableAndNoRecipe(t *testing.T) {
	db := openDB(t)
	if _, recorded, err := coarseRecipe(t.Context(), db.write); err != nil || recorded {
		t.Errorf("a fresh index records %v, %v", recorded, err)
	}
}
