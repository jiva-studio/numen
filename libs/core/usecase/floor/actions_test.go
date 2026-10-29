package floor_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/core/usecase/floor"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestSetValidatesAManualFloor(t *testing.T) {
	for _, tc := range []struct {
		name  string
		floor float64
		isOK  bool
	}{
		{"zero", 0, false},
		{"negative", -0.1, false},
		{"above one", 1.0001, false},
		{"not a number", math.NaN(), false},
		{"smallest above zero", 0.0001, true},
		{"one", 1, true},
		{"middle", 0.45, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newMemoryStore()
			err := floor.NewSet(store, &stubEmbedder{model: model}).Execute(t.Context(), vault, true, tc.floor)
			if tc.isOK != (err == nil) {
				t.Fatalf("got %v", err)
			}
			if !tc.isOK {
				if !errors.Is(err, floor.ErrInvalidFloor) || store.writes != 0 {
					t.Errorf("got %v with %d writes, want ErrInvalidFloor and none", err, store.writes)
				}
				return
			}
			got := store.settings[key(vault, model.Recipe())]
			if !got.IsManual || got.ManualFloor != tc.floor {
				t.Errorf("stored %+v", got)
			}
		})
	}
}

func TestSetClearsTheManualFloor(t *testing.T) {
	store := newMemoryStore()
	set := floor.NewSet(store, &stubEmbedder{model: model})
	if err := set.Execute(t.Context(), vault, true, 0.6); err != nil {
		t.Fatal(err)
	}
	// The value sent with a clear is not kept.
	if err := set.Execute(t.Context(), vault, false, 5); err != nil {
		t.Fatal(err)
	}
	if got := store.settings[key(vault, model.Recipe())]; got.IsManual || got.ManualFloor != 0 {
		t.Errorf("stored %+v, want the manual floor cleared", got)
	}
}

func TestSetNeedsAModel(t *testing.T) {
	if err := floor.NewSet(newMemoryStore(), nil).Execute(t.Context(), vault, true, 0.5); !errors.Is(err, floor.ErrNoModel) {
		t.Errorf("got %v, want ErrNoModel", err)
	}
}

// stubNearest answers every query with the same similarity.
type stubNearest struct {
	best   float64
	err    error
	recipe string
	query  []float32
}

func (n *stubNearest) GetBestSimilarity(_ context.Context, _ domain.VaultID, recipe string, query []float32) (float64, error) {
	n.recipe, n.query = recipe, query
	return n.best, n.err
}

func TestProbeComparesTheBestPassageWithTheFloor(t *testing.T) {
	for _, tc := range []struct {
		name     string
		best     float64
		wantPass bool
	}{
		{"below", 0.29, false},
		{"at the floor", 0.30, true},
		{"above", 0.8, true},
		{"nothing near", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nearest := &stubNearest{best: tc.best}
			embedder := unitEmbedder()
			probe := floor.NewProbe(nearest, floor.NewResolve(newMemoryStore(), 0, defaultFloor, nil, nil), embedder)

			got, err := probe.Execute(t.Context(), vault, "entropy")
			if err != nil {
				t.Fatal(err)
			}
			if got.BestSimilarity != tc.best || got.Floor != defaultFloor || got.DoesPass != tc.wantPass {
				t.Errorf("got %+v", got)
			}
			if nearest.recipe != model.Recipe() || len(nearest.query) != 4 {
				t.Errorf("asked under %q with %v", nearest.recipe, nearest.query)
			}
		})
	}
}

func TestProbeUsesTheFloorInForce(t *testing.T) {
	store := newMemoryStore()
	store.settings[key(vault, model.Recipe())] = floorport.Setting{IsManual: true, ManualFloor: 0.9}
	probe := floor.NewProbe(&stubNearest{best: 0.8}, floor.NewResolve(store, 0, defaultFloor, nil, nil), unitEmbedder())

	got, err := probe.Execute(t.Context(), vault, "entropy")
	if err != nil || got.Floor != 0.9 || got.DoesPass {
		t.Errorf("got %+v, %v, want floor 0.9 not passed", got, err)
	}
}

func TestProbeOfNoWordsEmbedsNothing(t *testing.T) {
	embedder := unitEmbedder()
	nearest := &stubNearest{best: 0.9}
	probe := floor.NewProbe(nearest, floor.NewResolve(newMemoryStore(), 0, defaultFloor, nil, nil), embedder)

	got, err := probe.Execute(t.Context(), vault, "   ")
	if err != nil || got.DoesPass || got.BestSimilarity != 0 || got.Floor != defaultFloor {
		t.Errorf("got %+v, %v", got, err)
	}
	if len(embedder.embedded) != 0 || nearest.recipe != "" {
		t.Errorf("an empty query reached the embedder or the index")
	}
}

func TestProbeErrors(t *testing.T) {
	resolve := floor.NewResolve(newMemoryStore(), 0, defaultFloor, nil, nil)
	if _, err := floor.NewProbe(&stubNearest{}, resolve, nil).Execute(t.Context(), vault, "q"); !errors.Is(err, floor.ErrNoModel) {
		t.Errorf("no embedder: got %v", err)
	}
	down := unitEmbedder()
	down.err = errPluginDown
	if _, err := floor.NewProbe(&stubNearest{}, resolve, down).Execute(t.Context(), vault, "q"); !errors.Is(err, floor.ErrEmbedding) || !errors.Is(err, errPluginDown) {
		t.Errorf("plugin down: got %v", err)
	}
	broken := errors.New("the index is locked")
	if _, err := floor.NewProbe(&stubNearest{err: broken}, resolve, unitEmbedder()).Execute(t.Context(), vault, "q"); !errors.Is(err, broken) {
		t.Errorf("index error: got %v", err)
	}
}

func newAuto(store *memoryStore, embedder *stubEmbedder, samples stubSamples) floor.AutoCalibrate {
	var calibrate floor.Calibrate
	if embedder == nil {
		calibrate = floor.NewCalibrate(samples, store, nil, newClock(calibratedAt))
		return floor.NewAutoCalibrate(store, calibrate, nil)
	}
	calibrate = floor.NewCalibrate(samples, store, embedder, newClock(calibratedAt))
	return floor.NewAutoCalibrate(store, calibrate, embedder)
}

func TestAutoCalibrateRunsOnlyWhereNothingIsStored(t *testing.T) {
	enough := stubSamples{titles: titlesOf(25), passages: passagesOf(60, 3)}
	for _, tc := range []struct {
		name      string
		setting   *floorport.Setting
		wantWrite bool
	}{
		{"no row", nil, true},
		{"a row with only a cleared manual floor", &floorport.Setting{}, true},
		{"a calibration", &floorport.Setting{Calibration: &floorport.Calibration{Floor: 0.4}}, false},
		{"a manual floor", &floorport.Setting{IsManual: true, ManualFloor: 0.5}, false},
		{"a manual floor and a calibration", &floorport.Setting{IsManual: true, ManualFloor: 0.5, Calibration: &floorport.Calibration{}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newMemoryStore()
			if tc.setting != nil {
				store.settings[key(vault, model.Recipe())] = *tc.setting
			}
			embedder := unitEmbedder()
			if err := newAuto(store, embedder, enough).Execute(t.Context(), vault); err != nil {
				t.Fatal(err)
			}
			if wrote := store.writes > 0; wrote != tc.wantWrite {
				t.Errorf("wrote = %v, want %v", wrote, tc.wantWrite)
			}
			if called := len(embedder.embedded) > 0; called != tc.wantWrite {
				t.Errorf("embedded = %v, want %v", called, tc.wantWrite)
			}
		})
	}
}

func TestAutoCalibrateRunsOncePerRecipe(t *testing.T) {
	store := newMemoryStore()
	embedder := unitEmbedder()
	auto := newAuto(store, embedder, stubSamples{titles: titlesOf(25), passages: passagesOf(60, 3)})
	for range 3 {
		if err := auto.Execute(t.Context(), vault); err != nil {
			t.Fatal(err)
		}
	}
	if len(embedder.embedded) != 1 {
		t.Errorf("calibrated %d times, want once", len(embedder.embedded))
	}
}

func TestAutoCalibrateIsSilentWhenTooFew(t *testing.T) {
	store := newMemoryStore()
	if err := newAuto(store, unitEmbedder(), stubSamples{titles: titlesOf(3), passages: passagesOf(10, 2)}).
		Execute(t.Context(), vault); err != nil {
		t.Errorf("got %v, want no error", err)
	}
	if store.writes != 0 {
		t.Errorf("%d writes, want none", store.writes)
	}
}

func TestAutoCalibrateReportsErrors(t *testing.T) {
	enough := stubSamples{titles: titlesOf(25), passages: passagesOf(60, 3)}
	down := unitEmbedder()
	down.err = errPluginDown
	if err := newAuto(newMemoryStore(), down, enough).Execute(t.Context(), vault); !errors.Is(err, errPluginDown) {
		t.Errorf("got %v, want the plugin's error", err)
	}

	store := newMemoryStore()
	store.readErr = errors.New("the index is locked")
	if err := newAuto(store, unitEmbedder(), enough).Execute(t.Context(), vault); !errors.Is(err, store.readErr) {
		t.Errorf("got %v, want the store's error", err)
	}
}

func TestAutoCalibrateWithNoEmbedderDoesNothing(t *testing.T) {
	store := newMemoryStore()
	if err := newAuto(store, nil, stubSamples{}).Execute(t.Context(), vault); err != nil || store.writes != 0 {
		t.Errorf("got %v with %d writes", err, store.writes)
	}
}

func TestNewScenariosShareOneStoreAndOneEmbedder(t *testing.T) {
	store := newMemoryStore()
	embedder := unitEmbedder()
	embedder.hint = 0.33
	scenarios := floor.NewScenarios(floor.Config{
		Store: store, Samples: stubSamples{titles: titlesOf(25), passages: passagesOf(60, 3)},
		Nearest: &stubNearest{best: 0.5}, Embedder: embedder, Clock: newClock(calibratedAt),
		DefaultFloor: defaultFloor, Hint: func(ctx context.Context) float64 { return embedder.GetFloorHint(ctx) },
	})

	if got := scenarios.Get.Execute(t.Context(), vault); got.Source != floor.SourceModel || got.Floor != 0.33 {
		t.Fatalf("before calibration: %+v", got)
	}
	if _, _, err := scenarios.Calibrate.Execute(t.Context(), vault); err != nil {
		t.Fatal(err)
	}
	if got := scenarios.Get.Execute(t.Context(), vault); got.Source != floor.SourceCalibrated || got.Calibration == nil {
		t.Errorf("after calibration: %+v", got)
	}
	if err := scenarios.Set.Execute(t.Context(), vault, true, 0.7); err != nil {
		t.Fatal(err)
	}
	if got := scenarios.Resolve.GetFloor(t.Context(), vault, model.Recipe()); got != 0.7 {
		t.Errorf("after a manual floor: %v", got)
	}
	if got, err := scenarios.Probe.Execute(t.Context(), vault, "q"); err != nil || got.Floor != 0.7 || got.DoesPass {
		t.Errorf("probe: %+v, %v", got, err)
	}
}
