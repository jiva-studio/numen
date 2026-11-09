package searchpolicy_test

import (
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// A vault id and a recipe with SQL in them are bound as values: the floor is stored under that
// exact text, and the table is still there afterwards.
func TestRepository_BindsValuesAsParameters(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db, err := index.Open(t.Context(), filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})

	vault := domain.Vault{ID: "x'; DROP TABLE search_floors; --"}
	recipe := "r' OR '1'='1"
	floors := db.SearchFloors()
	if err := floors.WriteFloor(t.Context(), vault, recipe, 0.25); err != nil {
		t.Fatal(err)
	}
	got, found, err := floors.ReadFloor(t.Context(), vault, recipe)
	if err != nil || !found || got != 0.25 {
		t.Fatalf("ReadFloor = %v, %v, %v; want 0.25, true, nil", got, found, err)
	}
	if _, found, err := floors.ReadFloor(t.Context(), domain.Vault{ID: "other"}, recipe); err != nil || found {
		t.Fatalf("ReadFloor of another vault = %v, %v; want false, nil", found, err)
	}
	if err := floors.DeleteFloor(t.Context(), vault, recipe); err != nil {
		t.Fatal(err)
	}
	if _, found, err := floors.ReadFloor(t.Context(), vault, recipe); err != nil || found {
		t.Fatal("the floor is still there after DeleteFloor, or the read failed")
	}
	if got, err := db.SearchSamples().ListSmallChunkVectors(t.Context(), vault, recipe, 5); err != nil || len(got) != 0 {
		t.Fatalf("ListSmallChunkVectors = %v, %v; want none", got, err)
	}
}
