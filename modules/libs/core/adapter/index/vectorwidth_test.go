package index

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// openBreakingDB is an index whose both pools go through the breaking driver,
// holding the first vault and nothing else.
func openBreakingDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "index.db")
	migrated.CopyTo(t, path)
	pool := openBreaking(t, path)
	db := &DB{write: pool, read: pool}
	if err := db.Vaults().Register(t.Context(), first.ID); err != nil {
		t.Fatal(err)
	}
	return db
}

// readWidth is the width the vector index holds and the recipe recorded as
// having filled it.
func readWidth(t *testing.T, db *DB) (int, string) {
	t.Helper()
	width, err := getVectorWidth(t.Context(), db.write)
	if err != nil {
		t.Fatal(err)
	}
	recipe, _, err := coarseRecipe(t.Context(), db.write)
	if err != nil {
		t.Fatal(err)
	}
	return width, recipe
}

// Fitting the vector index is all or nothing: a step that fails is reported, and
// the index keeps the width and the recipe it held.
func TestAFitThatFailsPreservesTheVectorIndexAsItWas(t *testing.T) {
	for _, step := range []struct {
		name   string
		breaks func(string) bool
	}{
		{"reading the width", containing("name = 'chunks_vec'")},
		{"reading the recipe", containing("SELECT recipe FROM coarse_recipe")},
		{"beginning the transaction", func(query string) bool { return query == "BEGIN" }},
		{"dropping the old index", containing("DROP TABLE IF EXISTS chunks_vec")},
		{"reading the vectors the recipe holds", containing("FROM vectors v JOIN chunks")},
		{"filling the index", containing("INSERT INTO chunks_vec")},
		{"recording the recipe", containing("INSERT INTO coarse_recipe")},
	} {
		t.Run(step.name, func(t *testing.T) {
			ctx := t.Context()
			db := openBreakingDB(t)
			if err := db.FitVectors(ctx, 384, narrowRecipe(384)); err != nil {
				t.Fatal(err)
			}
			embedAtWidth(t, db, first, 384)
			if err := db.FitVectors(ctx, 1024, narrowRecipe(1024)); err != nil {
				t.Fatal(err)
			}
			breakWhere(t, step.breaks)

			err := db.FitVectors(ctx, 384, narrowRecipe(384))
			stopBreaking()
			if !errors.Is(err, errBroken) {
				t.Fatalf("fitting answered %v", err)
			}
			width, recipe := readWidth(t, db)
			if width != 1024 || recipe != narrowRecipe(1024) {
				t.Errorf("the vector index holds width %d under %q, want 1024 under %q", width, recipe, narrowRecipe(1024))
			}
		})
	}
}

// held.
func TestAWidthOfNoDimensionsIsRefused(t *testing.T) {
	db := openDB(t)
	fit(t, db, testRecipe)

	for _, dims := range []int{0, -1} {
		err := db.FitVectors(t.Context(), dims, testRecipe)
		if err == nil || !strings.Contains(err.Error(), "is not a vector") {
			t.Errorf("a vector index of %d dimensions was answered %v", dims, err)
		}
	}
	if width, _ := readWidth(t, db); width != 1024 {
		t.Errorf("the vector index holds width %d, want 1024", width)
	}
}

// An index holding no vector index is given one of the width asked for.
func TestAnIndexWithoutAVectorIndexIsGivenOne(t *testing.T) {
	db := openDB(t)
	if _, err := db.write.ExecContext(t.Context(), `DROP TABLE chunks_vec`); err != nil {
		t.Fatal(err)
	}

	if err := db.FitVectors(t.Context(), 384, narrowRecipe(384)); err != nil {
		t.Fatal(err)
	}
	if width, _ := readWidth(t, db); width != 384 {
		t.Errorf("the vector index holds width %d, want 384", width)
	}
}

// A vector index whose width cannot be read is reported and left standing.
func TestAVectorIndexOfUnreadableWidthIsReported(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	for _, statement := range []string{
		`DROP TABLE chunks_vec`,
		`CREATE VIRTUAL TABLE chunks_vec USING vec0 (chunk_id integer primary key, coarse float[8])`,
	} {
		if _, err := db.write.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}

	if err := db.FitVectors(ctx, 384, narrowRecipe(384)); err == nil {
		t.Fatal("a vector index of unreadable width was fitted over")
	}
	var created string
	if err := db.write.QueryRowContext(ctx,
		`SELECT sql FROM sqlite_master WHERE name = 'chunks_vec'`).Scan(&created); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(created, "float[8]") {
		t.Errorf("the vector index was replaced by %s", created)
	}
}

// Every positive width is a vector, down to a single dimension.
func TestTheNarrowestWidthIsFitted(t *testing.T) {
	db := openDB(t)
	if err := db.FitVectors(t.Context(), 1, testRecipe); err != nil {
		t.Fatal(err)
	}
	if width, _ := readWidth(t, db); width != 1 {
		t.Errorf("the vector index holds width %d, want 1", width)
	}
}

// The width of an index holding no vector index is nothing.
func TestAnIndexWithoutAVectorIndexContainsNoWidth(t *testing.T) {
	db := openDB(t)
	if _, err := db.write.ExecContext(t.Context(), `DROP TABLE chunks_vec`); err != nil {
		t.Fatal(err)
	}
	width, err := getVectorWidth(t.Context(), db.write)
	if width != 0 || err != nil {
		t.Errorf("got %d, %v", width, err)
	}
}

// A width that cannot be read is nothing, and the report says why.
func TestAWidthThatCannotBeReadIsNothing(t *testing.T) {
	t.Run("the catalogue cannot be read", func(t *testing.T) {
		db := openBreakingDB(t)
		breakOn(t, "name = 'chunks_vec'")
		width, err := getVectorWidth(t.Context(), db.write)
		stopBreaking()
		if width != 0 || !errors.Is(err, errBroken) {
			t.Errorf("got %d, %v", width, err)
		}
	})
	t.Run("the table says nothing of its width", func(t *testing.T) {
		db := openDB(t)
		for _, statement := range []string{
			`DROP TABLE chunks_vec`,
			`CREATE VIRTUAL TABLE chunks_vec USING vec0 (chunk_id integer primary key, coarse float[8])`,
		} {
			if _, err := db.write.ExecContext(t.Context(), statement); err != nil {
				t.Fatal(err)
			}
		}
		width, err := getVectorWidth(t.Context(), db.write)
		if width != 0 || err == nil {
			t.Errorf("got %d, %v", width, err)
		}
	})
}

// A recipe that cannot be read is reported, with the recipe nothing and none
// recorded.
func TestARecipeThatCannotBeReadIsReported(t *testing.T) {
	db := openBreakingDB(t)
	breakOn(t, "SELECT recipe FROM coarse_recipe")
	recipe, recorded, err := coarseRecipe(t.Context(), db.write)
	stopBreaking()
	if !errors.Is(err, errBroken) || recipe != "" || recorded {
		t.Errorf("got %q, %v, %v", recipe, recorded, err)
	}
}

// Fitting reports the vectors it could not read, and leaves the index as it was.
func TestAFitThatCannotReadTheVectorsPreservesTheVectorIndexAsItWas(t *testing.T) {
	for _, c := range []struct {
		name       string
		unreadable bool
	}{
		{"a vector that cannot be taken", true},
		{"a read that ends in an error", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx := t.Context()
			db := openBreakingDB(t)
			if err := db.FitVectors(ctx, 384, narrowRecipe(384)); err != nil {
				t.Fatal(err)
			}
			embedAtWidth(t, db, first, 384)
			if err := db.FitVectors(ctx, 1024, narrowRecipe(1024)); err != nil {
				t.Fatal(err)
			}
			breakRows(t, "FROM vectors v JOIN chunks", c.unreadable)

			err := db.FitVectors(ctx, 384, narrowRecipe(384))
			if err == nil || !strings.Contains(err.Error(), "the vectors the recipe holds") {
				t.Fatalf("fitting answered %v", err)
			}
			if errors.Unwrap(err) == nil || (!c.unreadable && !errors.Is(err, errBroken)) {
				t.Errorf("the cause was lost: %v", err)
			}
			width, recipe := readWidth(t, db)
			if width != 1024 || recipe != narrowRecipe(1024) {
				t.Errorf("the vector index holds width %d under %q", width, recipe)
			}
		})
	}
}
