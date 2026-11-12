package index

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const otherRecipe = "other-model"

func saveSeededVectorsUnder(t *testing.T, db *DB, vault domain.Vault, recipe string, seed byte) int {
	t.Helper()
	ctx := t.Context()

	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(ctx, vault.ID, recipe, chunk.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	vectors := make([]chunk.Vector, 0, len(unembedded))
	for _, p := range unembedded {
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

func storedBits(t *testing.T, db *DB) map[int64][]byte {
	t.Helper()
	rows, err := db.read.QueryContext(t.Context(), `SELECT chunk_id, coarse FROM chunks_vec`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Error(err)
		}
	}()
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

func TestCoarseRecipe_SwitchBackRestoresFirstModelBits(t *testing.T) {
	db := openDB(t)
	cutBook(t, db, getFirstVault(), "library/tale.epub")
	fit(t, db, testRecipe)
	small := saveSeededVectorsUnder(t, db, getFirstVault(), testRecipe, 0x0f)

	fit(t, db, otherRecipe)
	if got := saveSeededVectorsUnder(t, db, getFirstVault(), otherRecipe, 0xf0); got != small {
		t.Fatalf("%d of %d chunks were unembedded under the other recipe", got, small)
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
	if unembedded := saveSeededVectorsUnder(t, db, getFirstVault(), testRecipe, 0xaa); unembedded != 0 {
		t.Errorf("%d chunks are unembedded under a recipe that holds every vector", unembedded)
	}
}

func TestFitRecipe_SameRecipeAndWidthKeepsCoarseIndex(t *testing.T) {
	db := openDB(t)
	book(t, db, getFirstVault(), "library/tale.epub", 0x0f)
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
	book(t, db, getFirstVault(), "library/tale.epub", 0x0f)
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
	book(t, db, getFirstVault(), "library/tale.epub", 0x0f)
	fit(t, db, testRecipe)

	fit(t, db, otherRecipe)
	if rows := len(storedBits(t, db)); rows != 0 {
		t.Errorf("%d coarse rows hold another recipe's bits", rows)
	}
	unembedded, err := db.ChunkVectors().GetUnembeddedChunks(t.Context(), getFirstVault().ID, otherRecipe, chunk.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) == 0 {
		t.Error("no chunk is unembedded under a recipe that holds no vector")
	}
}

func TestAFreshIndexContainsTheRecipeTableAndNoRecipe(t *testing.T) {
	db := openDB(t)
	if _, recorded, err := coarseRecipe(t.Context(), db.write); err != nil || recorded {
		t.Errorf("a fresh index records %v, %v", recorded, err)
	}
}

func TestACoarseQueryTheVectorsOfAnotherRecipeCannotAnswerIsReportedAsThat(t *testing.T) {
	db := openDB(t)
	book(t, db, getFirstVault(), "library/tale.epub", 0x0f)
	if err := db.FitVectors(t.Context(), 384, narrowRecipe(384)); err != nil {
		t.Fatal(err)
	}

	_, err := queriesOf(db).FindCoarseCandidates(t.Context(), getFirstVault().ID, sourcedomain.CoarseQuery{Recipe: testRecipe, Vector: getVectorForSeed(0x0f), Limit: 5})
	if !errors.Is(err, sourcedomain.ErrOtherRecipe) {
		t.Errorf("got %v, want the vectors to be reported as kept under another recipe", err)
	}
}
