package floor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/core/usecase/floor"
)

const defaultFloor = 0.30

func TestResolveFollowsTheOrderOfSources(t *testing.T) {
	recipe := model.Recipe()
	calibrated := &floorport.Calibration{Floor: 0.40}
	for _, tc := range []struct {
		name          string
		fileFloor     float64
		setting       *floorport.Setting
		hint          float64
		wantFloor     float64
		wantSource    floor.Source
		wantAutomatic float64
	}{
		{
			name:      "the settings file outranks the person's floor",
			fileFloor: 0.60, hint: 0.35,
			setting:   &floorport.Setting{IsManual: true, ManualFloor: 0.50, Calibration: calibrated},
			wantFloor: 0.60, wantSource: floor.SourceFile, wantAutomatic: 0.40,
		},
		{
			name: "the person's floor outranks the calibration",
			hint: 0.35, setting: &floorport.Setting{IsManual: true, ManualFloor: 0.50, Calibration: calibrated},
			wantFloor: 0.50, wantSource: floor.SourceManual, wantAutomatic: 0.40,
		},
		{
			name: "the calibration outranks the model's hint",
			hint: 0.35, setting: &floorport.Setting{IsManual: false, ManualFloor: 0.50, Calibration: calibrated},
			wantFloor: 0.40, wantSource: floor.SourceCalibrated, wantAutomatic: 0.40,
		},
		{
			name: "the model's hint outranks the default",
			hint: 0.35, wantFloor: 0.35, wantSource: floor.SourceModel, wantAutomatic: 0.35,
		},
		{
			name:      "the default answers last",
			wantFloor: defaultFloor, wantSource: floor.SourceDefault, wantAutomatic: defaultFloor,
		},
		{
			name: "a manual floor with no value is ignored",
			hint: 0.35, setting: &floorport.Setting{IsManual: true},
			wantFloor: 0.35, wantSource: floor.SourceModel, wantAutomatic: 0.35,
		},
		{
			name: "a hint above one is ignored",
			hint: 1.5, wantFloor: defaultFloor, wantSource: floor.SourceDefault, wantAutomatic: defaultFloor,
		},
		{
			name: "a negative hint is ignored",
			hint: -0.2, wantFloor: defaultFloor, wantSource: floor.SourceDefault, wantAutomatic: defaultFloor,
		},
		{
			name:      "the file with nothing else keeps the automatic floor the default",
			fileFloor: 0.55, wantFloor: 0.55, wantSource: floor.SourceFile, wantAutomatic: defaultFloor,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newMemoryStore()
			if tc.setting != nil {
				store.settings[key(vault, recipe)] = *tc.setting
			}
			hint := tc.hint
			resolve := floor.NewResolve(store, tc.fileFloor, defaultFloor,
				func(context.Context) float64 { return hint }, nil)

			got := resolve.Execute(t.Context(), vault, recipe)
			if got.Floor != tc.wantFloor || got.Source != tc.wantSource || got.AutomaticFloor != tc.wantAutomatic {
				t.Errorf("got floor %v from %v with automatic %v, want %v from %v with automatic %v",
					got.Floor, got.Source, got.AutomaticFloor, tc.wantFloor, tc.wantSource, tc.wantAutomatic)
			}
			if resolve.GetFloor(t.Context(), vault, recipe) != tc.wantFloor {
				t.Errorf("GetFloor disagrees with Execute")
			}
		})
	}
}

func TestResolveReportsWhatIsStored(t *testing.T) {
	store := newMemoryStore()
	calibration := &floorport.Calibration{Floor: 0.4, Median: 0.3}
	store.settings[key(vault, model.Recipe())] = floorport.Setting{IsManual: true, ManualFloor: 0.5, Calibration: calibration}

	got := floor.NewResolve(store, 0, defaultFloor, nil, nil).Execute(t.Context(), vault, model.Recipe())
	if !got.IsManual || got.ManualFloor != 0.5 || got.Calibration != calibration {
		t.Errorf("got %+v", got)
	}
}

func TestResolveIsPerVaultAndPerRecipe(t *testing.T) {
	store := newMemoryStore()
	store.settings[key(vault, "recipe-a")] = floorport.Setting{IsManual: true, ManualFloor: 0.7}
	resolve := floor.NewResolve(store, 0, defaultFloor, nil, nil)

	if got := resolve.GetFloor(t.Context(), vault, "recipe-a"); got != 0.7 {
		t.Errorf("the recipe that was set has floor %v, want 0.7", got)
	}
	if got := resolve.GetFloor(t.Context(), vault, "recipe-b"); got != defaultFloor {
		t.Errorf("another recipe has floor %v, want the default", got)
	}
	other := vault
	other.ID = "01OTHER"
	if got := resolve.GetFloor(t.Context(), other, "recipe-a"); got != defaultFloor {
		t.Errorf("another vault has floor %v, want the default", got)
	}
}

func TestResolveReportsAStoreItCannotReadAndGoesOn(t *testing.T) {
	store := newMemoryStore()
	store.readErr = errors.New("the index is locked")
	var reported []error
	resolve := floor.NewResolve(store, 0, defaultFloor,
		func(context.Context) float64 { return 0.35 }, func(err error) { reported = append(reported, err) })

	got := resolve.Execute(t.Context(), vault, model.Recipe())
	if got.Floor != 0.35 || got.Source != floor.SourceModel {
		t.Errorf("got %v from %v, want the hint", got.Floor, got.Source)
	}
	if len(reported) != 1 || !errors.Is(reported[0], store.readErr) {
		t.Errorf("reported %v", reported)
	}
}

func TestResolveWithNoStoreAnswersFromFileHintAndDefault(t *testing.T) {
	resolve := floor.NewResolve(nil, 0, defaultFloor, nil, nil)
	if got := resolve.Execute(t.Context(), vault, model.Recipe()); got.Floor != defaultFloor || got.Source != floor.SourceDefault {
		t.Errorf("got %+v", got)
	}
}

func TestResolveDoesNotReadTheStoreForNoRecipe(t *testing.T) {
	store := newMemoryStore()
	store.readErr = errors.New("must not be read")
	resolve := floor.NewResolve(store, 0, defaultFloor, nil, func(err error) { t.Errorf("reported %v", err) })
	if got := resolve.GetFloor(t.Context(), vault, ""); got != defaultFloor {
		t.Errorf("got %v", got)
	}
}

func TestGetNamesTheModelAndReadsItsRecipe(t *testing.T) {
	store := newMemoryStore()
	store.settings[key(vault, model.Recipe())] = floorport.Setting{IsManual: true, ManualFloor: 0.5}
	embedder := &stubEmbedder{model: model}
	get := floor.NewGet(floor.NewResolve(store, 0, defaultFloor, nil, nil), embedder)

	got := get.Execute(t.Context(), vault)
	if got.Model != "test-model" || got.Floor != 0.5 || got.Source != floor.SourceManual {
		t.Errorf("got %+v", got)
	}

	none := floor.NewGet(floor.NewResolve(store, 0, defaultFloor, nil, nil), nil).Execute(t.Context(), vault)
	if none.Model != "" || none.Floor != defaultFloor {
		t.Errorf("with no embedder got %+v", none)
	}
}
