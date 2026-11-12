package index

import (
	"math"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestSearchFloorsKeepOneFloorPerVaultAndRecipeAndReplaceIt(t *testing.T) {
	ctx := t.Context()
	repo := openDB(t).SearchFloors()

	if _, found, err := repo.ReadFloor(ctx, getFirstVault(), testRecipe); err != nil || found {
		t.Fatalf("a floor was found before any was written: %v, %v", found, err)
	}
	if err := repo.WriteFloor(ctx, getFirstVault(), testRecipe, 0.55); err != nil {
		t.Fatal(err)
	}
	if got, found, err := repo.ReadFloor(ctx, getFirstVault(), testRecipe); err != nil || !found || got != 0.55 {
		t.Fatalf("read after a write: %v, %v, %v", got, found, err)
	}
	if err := repo.WriteFloor(ctx, getFirstVault(), testRecipe, 0.61); err != nil {
		t.Fatal(err)
	}
	if got, _, err := repo.ReadFloor(ctx, getFirstVault(), testRecipe); err != nil || got != 0.61 {
		t.Errorf("a second write left %v, %v, want 0.61", got, err)
	}
	if err := repo.DeleteFloor(ctx, getFirstVault(), testRecipe); err != nil {
		t.Fatal(err)
	}
	if _, found, err := repo.ReadFloor(ctx, getFirstVault(), testRecipe); err != nil || found {
		t.Errorf("the floor is still kept after a delete: %v, %v", found, err)
	}
	if err := repo.DeleteFloor(ctx, getFirstVault(), testRecipe); err != nil {
		t.Errorf("deleting a floor that is not kept failed: %v", err)
	}
}

func TestSearchFloorsAreStoredPerVaultAndPerRecipe(t *testing.T) {
	ctx := t.Context()
	repo := openDB(t).SearchFloors()
	if err := repo.WriteFloor(ctx, getFirstVault(), testRecipe, 0.5); err != nil {
		t.Fatal(err)
	}
	if _, found, err := repo.ReadFloor(ctx, getSecond(), testRecipe); err != nil || found {
		t.Errorf("another vault found a floor: %v, %v", found, err)
	}
	if _, found, err := repo.ReadFloor(ctx, getFirstVault(), otherRecipe); err != nil || found {
		t.Errorf("another recipe found a floor: %v, %v", found, err)
	}
	if err := repo.DeleteFloor(ctx, getSecond(), testRecipe); err != nil {
		t.Fatal(err)
	}
	if _, found, err := repo.ReadFloor(ctx, getFirstVault(), testRecipe); err != nil || !found {
		t.Errorf("deleting the floor of another vault removed this one: %v, %v", found, err)
	}
}

func TestSearchFloorsSurviveReopeningTheIndex(t *testing.T) {
	ctx := t.Context()
	path := t.TempDir() + "/index.db"
	one := openDBAt(t, path)
	if err := one.SearchFloors().WriteFloor(ctx, getFirstVault(), testRecipe, 0.5); err != nil {
		t.Fatal(err)
	}
	if err := one.Close(); err != nil {
		t.Fatal(err)
	}
	two, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, two)
	got, found, err := two.SearchFloors().ReadFloor(ctx, getFirstVault(), testRecipe)
	if err != nil || !found || got != 0.5 {
		t.Errorf("after reopening: %v, %v, %v", got, found, err)
	}
}

func TestSearchSamplePassagesAreSmallChunksWithAVectorOfTheRecipe(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	book(t, db, getFirstVault(), "library/one.epub", 0x0f)
	book(t, db, getFirstVault(), "library/two.epub", 0xf0)
	book(t, db, getSecond(), "library/three.epub", 0xaa)
	cutBook(t, db, getFirstVault(), "library/bare.epub")

	samples := db.SearchSamples()
	got, err := samples.ListSmallChunkVectors(ctx, getFirstVault(), testRecipe, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d passages, want the 4 small chunks of the two embedded books", len(got))
	}
	sources := map[int64]int{}
	for _, passage := range got {
		sources[passage.Source]++
		if !strings.HasPrefix(passage.Path, "library/") {
			t.Errorf("a passage names the path %q, want one of the vault", passage.Path)
		}
		if len(passage.Vector) != 1024 {
			t.Errorf("a passage holds %d dimensions, want 1024", len(passage.Vector))
		}
	}
	if len(sources) != 2 {
		t.Errorf("the passages come from %d sources, want 2", len(sources))
	}

	other, err := samples.ListSmallChunkVectors(ctx, getFirstVault(), otherRecipe, 100)
	if err != nil || len(other) != 0 {
		t.Errorf("another recipe sampled %d passages: %v", len(other), err)
	}

	limited, err := samples.ListSmallChunkVectors(ctx, getFirstVault(), testRecipe, 3)
	if err != nil || len(limited) != 3 {
		t.Errorf("a limit of 3 sampled %d passages: %v", len(limited), err)
	}
	again, err := samples.ListSmallChunkVectors(ctx, getFirstVault(), testRecipe, 100)
	if err != nil {
		t.Fatal(err)
	}
	for i := range got {
		if got[i].Source != again[i].Source {
			t.Errorf("two reads ordered the passages differently at %d", i)
		}
	}
}

func TestGetBestSimilarityIsTheNearestPassageOfTheVault(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	book(t, db, getFirstVault(), "library/one.epub", 0x00)
	book(t, db, getSecond(), "library/two.epub", 0xff)
	queries := queriesOf(db)

	near, err := queries.GetBestSimilarity(ctx, getFirstVault().ID, testRecipe, getVectorForSeed(0x00))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(near-1) > 0.01 {
		t.Errorf("a vector the vault holds has similarity %v, want 1", near)
	}

	opposite, err := queries.GetBestSimilarity(ctx, getFirstVault().ID, testRecipe, getVectorForSeed(0xff))
	if err != nil {
		t.Fatal(err)
	}
	if opposite > -0.9 {
		t.Errorf("the opposite vector has similarity %v, want -1", opposite)
	}

	for name, probe := range map[string]struct {
		recipe string
		query  []float32
	}{
		"another recipe": {otherRecipe, getVectorForSeed(0x00)},
		"no vector":      {testRecipe, nil},
	} {
		got, err := queries.GetBestSimilarity(ctx, getFirstVault().ID, probe.recipe, probe.query)
		if err != nil || got != 0 {
			t.Errorf("%s: got %v, %v, want 0", name, got, err)
		}
	}
}
