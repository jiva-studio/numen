package relevance_test

import (
	"errors"
	"math"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/relevance"
)

// newFloors is the floors over a host that shows an embedded vault of 25 notes, and the vault whose
// store folder holds the records.
func newFloors(t *testing.T, hint float64) (*relevance.Floors, *hosttest.Search, *hosttest.Vault) {
	t.Helper()
	search, vault := newHost(hint)
	fill(search, vault, 25, passagesOf(60, 3))
	return relevance.New(relevance.Host{Search: search, Vault: vault, Store: vault.Store()}, newClock()), search, vault
}

func TestGetShowsTheHintOfTheModelBeforeAnythingIsStored(t *testing.T) {
	floors, _, _ := newFloors(t, 0.4)
	got, err := floors.Get(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != relevance.SourceModel || got.Floor != 0.4 || got.AutomaticFloor != 0.4 || got.Model != modelName ||
		got.IsManual || got.Calibration != nil {
		t.Errorf("got %+v", got)
	}
}

func TestGetShowsTheDefaultWhereTheModelHasNoHint(t *testing.T) {
	floors, _, _ := newFloors(t, 0)
	got, err := floors.Get(t.Context(), "")
	if err != nil || got.Source != relevance.SourceDefault || got.Floor != 0.30 || got.AutomaticFloor != 0.30 {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestGetShowsTheDefaultWhereTheInstallationHasNoModel(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	search.RemoveModel()
	got, err := floors.Get(t.Context(), "")
	if err != nil || got.Source != relevance.SourceDefault || got.Floor != 0.30 || got.Model != "" {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestCalibrateKeepsTheMeasurementAndTheHostAppliesItsFloor(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)

	outcome, state, err := floors.Calibrate(t.Context(), "")
	if err != nil || outcome != relevance.Calibrated {
		t.Fatalf("got %v, %v", outcome, err)
	}
	if state.Source != relevance.SourceCalibrated || abs(state.Floor-0.52) > 1e-6 || state.Calibration == nil ||
		state.Calibration.QueryCount != 25 || state.IsManual {
		t.Errorf("state %+v", state)
	}
	if stored, found := search.StoredFloor(modelRecipe); !found || stored != state.Floor {
		t.Errorf("the host stores %v (%v), want %v", stored, found, state.Floor)
	}

	again, err := floors.Get(t.Context(), "")
	if err != nil || again.Calibration == nil || again.Source != relevance.SourceCalibrated || again.Floor != state.Floor {
		t.Errorf("read again: %+v, %v", again, err)
	}
}

func TestCalibrateOfAVaultWithTooFewNotesChangesNothing(t *testing.T) {
	floors, search, vault := newFloors(t, 0.4)
	fill(search, vault, 5, passagesOf(10, 2))

	outcome, state, err := floors.Calibrate(t.Context(), "")
	if err != nil || outcome != relevance.TooFew {
		t.Fatalf("got %v, %v", outcome, err)
	}
	if state.Source != relevance.SourceModel || state.Calibration != nil {
		t.Errorf("state %+v", state)
	}
	if _, found := search.StoredFloor(modelRecipe); found || search.FloorWrites() != 0 {
		t.Error("a floor was stored for a vault too small to measure")
	}
}

func TestCalibrateRespectsTheFloorOfThePerson(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	if _, err := floors.Set(t.Context(), "", true, 0.7); err != nil {
		t.Fatal(err)
	}

	_, state, err := floors.Calibrate(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	if state.Floor != 0.7 || state.Source != relevance.SourceManual || !state.IsManual || state.ManualFloor != 0.7 {
		t.Errorf("state %+v, want the manual floor in force", state)
	}
	if abs(state.AutomaticFloor-0.52) > 1e-6 || state.Calibration == nil {
		t.Errorf("state %+v, want the measurement kept beside it", state)
	}
	if stored, _ := search.StoredFloor(modelRecipe); stored != 0.7 {
		t.Errorf("the host applies %v, want 0.7", stored)
	}
}

func TestSetValidatesTheFloorOfThePerson(t *testing.T) {
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
			floors, search, _ := newFloors(t, 0.4)
			state, err := floors.Set(t.Context(), "", true, tc.floor)
			if tc.isOK != (err == nil) {
				t.Fatalf("got %v", err)
			}
			if !tc.isOK {
				if !errors.Is(err, host.ErrInvalidFloor) || search.FloorWrites() != 0 {
					t.Errorf("got %v with %d writes, want ErrInvalidFloor and none", err, search.FloorWrites())
				}
				return
			}
			if !state.IsManual || state.ManualFloor != tc.floor || state.Floor != tc.floor || state.Source != relevance.SourceManual {
				t.Errorf("state %+v", state)
			}
		})
	}
}

func TestSetOfAutomaticReturnsToTheMeasuredFloorAndDropsTheValueSent(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	if _, _, err := floors.Calibrate(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := floors.Set(t.Context(), "", true, 0.6); err != nil {
		t.Fatal(err)
	}

	state, err := floors.Set(t.Context(), "", false, 5)
	if err != nil {
		t.Fatal(err)
	}
	if state.IsManual || state.ManualFloor != 0 || state.Source != relevance.SourceCalibrated || abs(state.Floor-0.52) > 1e-6 {
		t.Errorf("state %+v, want the measured floor and no manual floor", state)
	}
	if stored, _ := search.StoredFloor(modelRecipe); abs(stored-0.52) > 1e-6 {
		t.Errorf("the host applies %v, want the measured floor", stored)
	}
}

func TestSetOfAutomaticWithoutAMeasurementFallsBackToTheHint(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	if _, err := floors.Set(t.Context(), "", true, 0.6); err != nil {
		t.Fatal(err)
	}

	state, err := floors.Set(t.Context(), "", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if state.Floor != 0.4 || state.Source != relevance.SourceModel {
		t.Errorf("state %+v, want the hint", state)
	}
	if _, found := search.StoredFloor(modelRecipe); found {
		t.Error("the host still stores a floor")
	}
}

func TestSetNeedsAModel(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	search.RemoveModel()
	if _, err := floors.Set(t.Context(), "", true, 0.5); !errors.Is(err, host.ErrNoModel) {
		t.Errorf("got %v, want ErrNoModel", err)
	}
}

func TestTheRecordsAreStoredPerVaultAndPerRecipe(t *testing.T) {
	floors, search, vault := newFloors(t, 0.4)
	if _, err := floors.Set(t.Context(), "", true, 0.6); err != nil {
		t.Fatal(err)
	}

	other, err := floors.Get(t.Context(), "chemistry")
	if err != nil || other.IsManual || other.Floor != 0.4 {
		t.Errorf("another vault read %+v, %v, want nothing kept", other, err)
	}

	search.SetModel(hosttest.SearchModel{Name: "other", Recipe: "other|recipe", FloorHint: 0.35})
	changed, err := floors.Get(t.Context(), "")
	if err != nil || changed.IsManual || changed.Floor != 0.35 || changed.Model != "other" {
		t.Errorf("another model read %+v, %v, want its own hint", changed, err)
	}

	search.SetModel(hosttest.SearchModel{Name: modelName, Recipe: modelRecipe, FloorHint: 0.4})
	back, err := floors.Get(t.Context(), "")
	if err != nil || !back.IsManual || back.ManualFloor != 0.6 {
		t.Errorf("the first model read %+v, %v, want its manual floor again", back, err)
	}
	if files, err := vault.Store().List(t.Context()); err != nil || len(files) != 1 {
		t.Errorf("the folder holds %v (%v), want one record", files, err)
	}
}

func TestAnUnreadableRecordIsAnErrorAndNotAnEmptyOne(t *testing.T) {
	floors, _, vault := newFloors(t, 0.4)
	if _, err := floors.Set(t.Context(), "", true, 0.6); err != nil {
		t.Fatal(err)
	}
	files, err := vault.Store().List(t.Context())
	if err != nil || len(files) != 1 {
		t.Fatalf("listed %v (%v)", files, err)
	}
	vault.Store().File(files[0].Name, []byte("{broken"))

	if _, err := floors.Get(t.Context(), ""); err == nil {
		t.Error("a broken record was read as an empty one")
	}
}

func TestProbeComparesTheBestPassageWithTheFloorInForce(t *testing.T) {
	for _, tc := range []struct {
		name     string
		stored   []float32
		wantBest float64
		wantPass bool
	}{
		{"below", []float32{30, 127, 0, 0}, 0.23, false},
		{"above", []float32{127, 0, 0, 0}, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			floors, search, _ := newFloors(t, 0)
			search.SetPassages(modelRecipe, []hosttest.SearchPassage{{Path: notePath(1), Vector: tc.stored}})

			got, err := floors.Probe(t.Context(), "", "entropy")
			if err != nil {
				t.Fatal(err)
			}
			if abs(got.BestSimilarity-tc.wantBest) > 0.01 || got.Floor != 0.30 || got.DoesPass != tc.wantPass {
				t.Errorf("got %+v", got)
			}
		})
	}
}

func TestProbeUsesTheFloorInForce(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	search.SetPassages(modelRecipe, []hosttest.SearchPassage{{Path: notePath(1), Vector: []float32{127, 20, 0, 0}}})
	if _, err := floors.Set(t.Context(), "", true, 0.99); err != nil {
		t.Fatal(err)
	}

	got, err := floors.Probe(t.Context(), "", "entropy")
	if err != nil || got.Floor != 0.99 || got.DoesPass {
		t.Errorf("got %+v, %v, want floor 0.99 not passed", got, err)
	}
}

func TestProbeOfNoWordsEmbedsNothing(t *testing.T) {
	floors, search, _ := newFloors(t, 0)
	got, err := floors.Probe(t.Context(), "", "   ")
	if err != nil || got.DoesPass || got.BestSimilarity != 0 || got.Floor != 0.30 {
		t.Errorf("got %+v, %v", got, err)
	}
	if batches := search.EmbeddedBatches(); len(batches) != 0 {
		t.Error("an empty query reached the model")
	}
}

func TestProbeReportsTheFailuresOfTheHost(t *testing.T) {
	floors, search, _ := newFloors(t, 0)
	search.FailEmbedding(errors.New("the plugin is not running"))
	if _, err := floors.Probe(t.Context(), "", "q"); !errors.Is(err, host.ErrEmbeddingFailed) {
		t.Errorf("a model that fails gave %v", err)
	}
	search.RemoveModel()
	if _, err := floors.Probe(t.Context(), "", "q"); !errors.Is(err, host.ErrNoModel) {
		t.Errorf("no model gave %v", err)
	}
}

func TestEnsureMeasuresAnEmbeddedVaultOnceAndWritesTheFloorOnce(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	for range 3 {
		if err := floors.Ensure(t.Context(), "physics"); err != nil {
			t.Fatal(err)
		}
	}
	if batches := search.EmbeddedBatches(); len(batches) != 1 {
		t.Errorf("the vault was measured %d times, want once", len(batches))
	}
	if stored, found := search.StoredFloor(modelRecipe); !found || abs(stored-0.52) > 1e-6 {
		t.Errorf("the host applies %v (%v), want the measured floor", stored, found)
	}
	if search.FloorWrites() != 1 {
		t.Errorf("the floor was written %d times, want once", search.FloorWrites())
	}
}

func TestEnsureWaitsForTheVectors(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	search.SetProgress(60, 20)
	if err := floors.Ensure(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	if batches := search.EmbeddedBatches(); len(batches) != 0 {
		t.Error("a vault still being embedded was measured")
	}
	search.SetProgress(60, 60)
	if err := floors.Ensure(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	if _, found := search.StoredFloor(modelRecipe); !found {
		t.Error("the vault was not measured once the vectors existed")
	}
}

func TestEnsureTreatsAStartAfterTheVectorsAsTheEventItMissed(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	search.SetProgress(60, 60)
	if err := floors.Ensure(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	if _, found := search.StoredFloor(modelRecipe); !found {
		t.Error("a plugin that started after the vectors existed did not measure the vault")
	}
}

func TestEnsureOfAVaultWithTooFewNotesTriesAgainAtTheNextCall(t *testing.T) {
	floors, search, vault := newFloors(t, 0.4)
	fill(search, vault, 5, passagesOf(10, 2))
	if err := floors.Ensure(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	if search.FloorWrites() != 0 {
		t.Error("a floor was written for a vault too small to measure")
	}
	fill(search, vault, 25, passagesOf(60, 3))
	if err := floors.Ensure(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	if _, found := search.StoredFloor(modelRecipe); !found {
		t.Error("the vault was not measured once it was large enough")
	}
}

func TestEnsureAppliesTheRecordWhereTheHostLostItsFloor(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	if _, err := floors.Set(t.Context(), "", true, 0.66); err != nil {
		t.Fatal(err)
	}
	if err := search.SetSearchFloor(t.Context(), modelRecipe, 0); err != nil {
		t.Fatal(err)
	}

	if err := floors.Ensure(t.Context(), ""); err != nil {
		t.Fatal(err)
	}
	if stored, _ := search.StoredFloor(modelRecipe); stored != 0.66 {
		t.Errorf("the host applies %v, want the floor of the record, 0.66", stored)
	}
	if batches := search.EmbeddedBatches(); len(batches) != 0 {
		t.Error("a vault with a record was measured again")
	}
}

func TestEnsureDoesNothingWithoutAModel(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	search.RemoveModel()
	if err := floors.Ensure(t.Context(), ""); err != nil {
		t.Errorf("got %v, want no error", err)
	}
}

func TestEnsureReportsTheFailuresOfTheHost(t *testing.T) {
	floors, search, _ := newFloors(t, 0.4)
	search.FailEmbedding(errors.New("the plugin is not running"))
	if err := floors.Ensure(t.Context(), ""); !errors.Is(err, host.ErrEmbeddingFailed) {
		t.Errorf("got %v, want the failure of the model", err)
	}
	if search.FloorWrites() != 0 {
		t.Error("a floor was written after a failed measurement")
	}
}

func TestACallOnAnUnknownVaultIsRefused(t *testing.T) {
	floors, _, _ := newFloors(t, 0.4)
	if _, err := floors.Get(t.Context(), "biology"); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("got %v, want ErrUnknownVault", err)
	}
	if err := floors.Ensure(t.Context(), "biology"); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("got %v, want ErrUnknownVault", err)
	}
}
