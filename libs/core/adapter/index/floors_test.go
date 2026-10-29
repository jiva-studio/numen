package index

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/floorport"
)

func TestMigrationEightCreatesTheFloorsTable(t *testing.T) {
	available, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(available, func(m migration) bool { return m.version == 8 && m.name == "0008_floor.sql" }) {
		t.Fatal("migration 0008_floor.sql is not embedded")
	}

	db := openDB(t)
	columns := map[string]bool{}
	rows, err := db.read.QueryContext(t.Context(), `SELECT name FROM pragma_table_info('floors')`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"vault", "recipe", "is_manual", "manual_floor", "floor", "median", "spread",
		"histogram", "query_count", "passage_count", "calibrated_at",
	} {
		if !columns[want] {
			t.Errorf("the floors table has no column %s", want)
		}
	}
	if got := getNewestVersion(t); got < 8 {
		t.Errorf("newest version = %d, want at least 8", got)
	}
}

func TestFloorsKeepACalibrationAndAManualFloorApart(t *testing.T) {
	ctx := t.Context()
	repo := openDB(t).Floors()
	calibration := floorport.Calibration{
		Floor: 0.42, Median: 0.30, Spread: 0.06, Histogram: []int{0, 3, 9, 4},
		QueryCount: 30, PassageCount: 80, CalibratedTime: time.Unix(1_700_000_000, 0).UTC(),
	}

	if _, found, err := repo.ReadFloor(ctx, first, testRecipe); err != nil || found {
		t.Fatalf("a floor was found before any was written: %v, %v", found, err)
	}

	if err := repo.WriteManualFloor(ctx, first, testRecipe, true, 0.55); err != nil {
		t.Fatal(err)
	}
	got, found, err := repo.ReadFloor(ctx, first, testRecipe)
	if err != nil || !found {
		t.Fatalf("read after a manual write: %v, %v", found, err)
	}
	if !got.IsManual || got.ManualFloor != 0.55 || got.Calibration != nil {
		t.Errorf("manual floor alone read back as %+v", got)
	}

	if err := repo.WriteCalibration(ctx, first, testRecipe, calibration); err != nil {
		t.Fatal(err)
	}
	got, _, err = repo.ReadFloor(ctx, first, testRecipe)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsManual || got.ManualFloor != 0.55 {
		t.Errorf("a calibration changed the manual floor: %+v", got)
	}
	if got.Calibration == nil || !calibrationsEqual(*got.Calibration, calibration) {
		t.Errorf("calibration read back as %+v, want %+v", got.Calibration, calibration)
	}

	if err := repo.WriteManualFloor(ctx, first, testRecipe, false, 0); err != nil {
		t.Fatal(err)
	}
	got, _, err = repo.ReadFloor(ctx, first, testRecipe)
	if err != nil {
		t.Fatal(err)
	}
	if got.IsManual || got.ManualFloor != 0 {
		t.Errorf("manual floor not cleared: %+v", got)
	}
	if got.Calibration == nil || !calibrationsEqual(*got.Calibration, calibration) {
		t.Errorf("clearing the manual floor changed the calibration: %+v", got.Calibration)
	}
}

func TestFloorsAreStoredPerVaultAndPerRecipe(t *testing.T) {
	ctx := t.Context()
	repo := openDB(t).Floors()
	if err := repo.WriteManualFloor(ctx, first, testRecipe, true, 0.5); err != nil {
		t.Fatal(err)
	}
	for name, probe := range map[string]struct {
		vault  string
		recipe string
	}{
		"another vault":  {vault: "second", recipe: testRecipe},
		"another recipe": {vault: "first", recipe: otherRecipe},
	} {
		v := first
		if probe.vault == "second" {
			v = second
		}
		if _, found, err := repo.ReadFloor(ctx, v, probe.recipe); err != nil || found {
			t.Errorf("%s found a floor: %v, %v", name, found, err)
		}
	}
}

func TestFloorsSurviveReopeningTheIndex(t *testing.T) {
	ctx := t.Context()
	path := t.TempDir() + "/index.db"
	one := openDBAt(t, path)
	if err := one.Floors().WriteManualFloor(ctx, first, testRecipe, true, 0.5); err != nil {
		t.Fatal(err)
	}
	if err := one.Close(); err != nil {
		t.Fatal(err)
	}
	two, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer two.Close()
	got, found, err := two.Floors().ReadFloor(ctx, first, testRecipe)
	if err != nil || !found || !got.IsManual || got.ManualFloor != 0.5 {
		t.Errorf("after reopening: %+v, %v, %v", got, found, err)
	}
}

func calibrationsEqual(a, b floorport.Calibration) bool {
	return a.Floor == b.Floor && a.Median == b.Median && a.Spread == b.Spread &&
		slices.Equal(a.Histogram, b.Histogram) && a.QueryCount == b.QueryCount &&
		a.PassageCount == b.PassageCount && a.CalibratedTime.Equal(b.CalibratedTime)
}

func TestFloorSamplesAreScopedToTheVaultAndOrderedByContent(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	for _, title := range []string{"Entropy", "Gravity", "Harmony", "Tide"} {
		if err := saveNote(t, db, first, "notes/"+title+".md", title); err != nil {
			t.Fatal(err)
		}
	}
	if err := saveNote(t, db, second, "notes/Other.md", "Other"); err != nil {
		t.Fatal(err)
	}

	samples := db.FloorSamples()
	all, err := samples.GetTitles(ctx, first, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("got %d titles, want the 4 of the first vault: %+v", len(all), all)
	}
	for _, title := range all {
		if title.Text == "Other" {
			t.Errorf("the title of another vault was sampled")
		}
	}

	again, err := samples.GetTitles(ctx, first, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(all, again) {
		t.Errorf("two reads gave %+v and %+v", all, again)
	}

	few, err := samples.GetTitles(ctx, first, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(few, all[:2]) {
		t.Errorf("a limit of 2 gave %+v, want the first two of %+v", few, all)
	}
}

func TestFloorSamplePassagesAreSmallChunksWithAVectorOfTheRecipe(t *testing.T) {
	ctx := t.Context()
	db := openDB(t)
	book(t, db, first, "library/one.epub", 0x0f)
	book(t, db, first, "library/two.epub", 0xf0)
	book(t, db, second, "library/three.epub", 0xaa)
	cutBook(t, db, first, "library/bare.epub")

	samples := db.FloorSamples()
	got, err := samples.GetPassages(ctx, first, testRecipe, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d passages, want the 4 small chunks of the two embedded books", len(got))
	}
	sources := map[int64]int{}
	for _, passage := range got {
		sources[passage.Source]++
		if len(passage.Vector) != 1024 {
			t.Errorf("a passage holds %d dimensions, want 1024", len(passage.Vector))
		}
	}
	if len(sources) != 2 {
		t.Errorf("the passages come from %d sources, want 2", len(sources))
	}

	other, err := samples.GetPassages(ctx, first, otherRecipe, 100)
	if err != nil || len(other) != 0 {
		t.Errorf("another recipe sampled %d passages: %v", len(other), err)
	}

	limited, err := samples.GetPassages(ctx, first, testRecipe, 3)
	if err != nil || len(limited) != 3 {
		t.Errorf("a limit of 3 sampled %d passages: %v", len(limited), err)
	}
	again, err := samples.GetPassages(ctx, first, testRecipe, 100)
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
	book(t, db, first, "library/one.epub", 0x00)
	book(t, db, second, "library/two.epub", 0xff)
	queries := db.ChunkQueries()

	near, err := queries.GetBestSimilarity(ctx, first.ID, testRecipe, getVectorForSeed(0x00))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(near-1) > 0.01 {
		t.Errorf("a vector the vault holds has similarity %v, want 1", near)
	}

	opposite, err := queries.GetBestSimilarity(ctx, first.ID, testRecipe, getVectorForSeed(0xff))
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
		got, err := queries.GetBestSimilarity(ctx, first.ID, probe.recipe, probe.query)
		if err != nil || got != 0 {
			t.Errorf("%s: got %v, %v, want 0", name, got, err)
		}
	}
}
