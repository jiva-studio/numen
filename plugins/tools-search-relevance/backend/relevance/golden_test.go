package relevance_test

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/relevance"
)

// goldenFixture is a sample vault and the calibration the calibration of the core measured on it
// before that code moved into this plugin. The samples are the titles with their vectors and the
// stored vectors of the passages. The expected values are the output of the core's
// usecase/floor Calibrate on those samples, so the plugin has to measure the same floor.
type goldenFixture struct {
	Name   string `json:"name"`
	Titles []struct {
		Source int64     `json:"source"`
		Text   string    `json:"text"`
		Vector []float32 `json:"vector"`
	} `json:"titles"`
	Passages []struct {
		Source int64  `json:"source"`
		Vector []byte `json:"vector"`
	} `json:"passages"`
	Expected struct {
		Outcome      string  `json:"outcome"`
		Floor        float64 `json:"floor"`
		Median       float64 `json:"median"`
		Spread       float64 `json:"spread"`
		Histogram    []int   `json:"histogram"`
		QueryCount   int     `json:"queryCount"`
		PassageCount int     `json:"passageCount"`
	} `json:"expected"`
}

func readFixtures(t *testing.T) []goldenFixture {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no golden fixture in testdata: %v", err)
	}
	fixtures := make([]goldenFixture, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fixture goldenFixture
		if err := json.Unmarshal(raw, &fixture); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures
}

// hostOf is a host whose index holds the samples of the fixture, in the order they were recorded.
func hostOf(fixture goldenFixture) *hosttest.Search {
	search := hosttest.NewSearch("physics")
	search.SetModel(hosttest.SearchModel{Name: modelName, Recipe: modelRecipe})
	vectors := map[string][]float32{}
	titles := make([]host.NoteTitle, len(fixture.Titles))
	for i, one := range fixture.Titles {
		vectors[one.Text] = one.Vector
		titles[i] = host.NoteTitle{Source: one.Source, Title: one.Text}
	}
	search.SetEmbedder(func(text string) []float32 { return vectors[text] })
	search.SetNoteTitles(titles)
	passages := make([]host.ChunkVector, len(fixture.Passages))
	for i, one := range fixture.Passages {
		passages[i] = host.ChunkVector{Source: one.Source, Vector: one.Vector}
	}
	search.SetChunkVectors(modelRecipe, passages)
	return search
}

func TestTheSamplesOfTheCoreGiveTheSameFloorInThePlugin(t *testing.T) {
	for _, fixture := range readFixtures(t) {
		t.Run(fixture.Name, func(t *testing.T) {
			search := hostOf(fixture)
			state, err := search.GetEmbeddingState(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			outcome, got, err := relevance.NewCalibrator(newClock()).Calibrate(t.Context(), search, state)
			if err != nil {
				t.Fatal(err)
			}
			want := fixture.Expected
			if want.Outcome == "too-few" {
				if outcome != relevance.TooFew {
					t.Fatalf("outcome %v, want TooFew", outcome)
				}
				return
			}
			if outcome != relevance.Calibrated {
				t.Fatalf("outcome %v, want Calibrated", outcome)
			}
			const tolerance = 1e-9
			if math.Abs(got.Floor-want.Floor) > tolerance || math.Abs(got.Median-want.Median) > tolerance ||
				math.Abs(got.Spread-want.Spread) > tolerance {
				t.Errorf("floor %v median %v spread %v, want %v, %v and %v",
					got.Floor, got.Median, got.Spread, want.Floor, want.Median, want.Spread)
			}
			if !slices.Equal(got.Histogram, want.Histogram) {
				t.Errorf("histogram %v, want %v", got.Histogram, want.Histogram)
			}
			if got.QueryCount != want.QueryCount || got.PassageCount != want.PassageCount {
				t.Errorf("measured %d queries against %d passages, want %d against %d",
					got.QueryCount, got.PassageCount, want.QueryCount, want.PassageCount)
			}
		})
	}
}

func TestTheFloorOfTheCoreIsStoredByTheFloorsOfThePlugin(t *testing.T) {
	for _, fixture := range readFixtures(t) {
		if fixture.Expected.Outcome == "too-few" {
			continue
		}
		t.Run(fixture.Name, func(t *testing.T) {
			search := hostOf(fixture)
			search.SetProgress(10, 10)
			vault := hosttest.New("physics")
			floors := relevance.New(relevance.Host{Search: search, Store: vault.Store()}, newClock())

			outcome, state, err := floors.Calibrate(t.Context(), "")
			if err != nil || outcome != relevance.Calibrated {
				t.Fatalf("got %v, %v", outcome, err)
			}
			stored, found := search.StoredFloor(modelRecipe)
			if !found || math.Abs(stored-fixture.Expected.Floor) > 1e-9 {
				t.Errorf("the host stores %v (%v), want %v", stored, found, fixture.Expected.Floor)
			}
			if state.Source != relevance.SourceCalibrated || math.Abs(state.Floor-fixture.Expected.Floor) > 1e-9 {
				t.Errorf("state %+v, want the calibrated floor %v", state, fixture.Expected.Floor)
			}
		})
	}
}
