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
	width, err := vectorWidth(t.Context(), db.write)
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
func TestAFitThatFailsLeavesTheVectorIndexAsItWas(t *testing.T) {
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
			// Vectors of 384 dimensions are held, and the index stands at 1024.
			if err := db.FitVectors(ctx, 384, narrowRecipe(384)); err != nil {
				t.Fatal(err)
			}
			narrow(t, db, first, 384)
			if err := db.FitVectors(ctx, 1024, narrowRecipe(1024)); err != nil {
				t.Fatal(err)
			}
			breakWhere(t, step.breaks)

			err := db.FitVectors(ctx, 384, narrowRecipe(384))
			mend()
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

// A width of no dimensions is refused, and the vector index keeps the one it
// held.
func TestAWidthOfNoDimensionsIsRefused(t *testing.T) {
	db := openDB(t)
	fit(t, db, testRecipe)

	for _, dims := range []int{0, -1} {
		if err := db.FitVectors(t.Context(), dims, testRecipe); err == nil {
			t.Errorf("a vector index of %d dimensions was made", dims)
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
