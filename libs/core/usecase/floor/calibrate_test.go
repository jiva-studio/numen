package floor_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/core/usecase/floor"
)

var calibratedAt = time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)

// titlesOf is n titles, each on a source of its own numbered from 1.
func titlesOf(n int) []floorport.Title {
	out := make([]floorport.Title, n)
	for i := range out {
		out[i] = floorport.Title{Source: int64(i + 1), Text: fmt.Sprintf("title %d", i)}
	}
	return out
}

// passagesOf is n passages spread over `sources` sources numbered from 100, each
// at 0.5 cosine similarity to the unit vector along the first dimension.
func passagesOf(n, sources int) []floorport.Passage {
	out := make([]floorport.Passage, n)
	for i := range out {
		out[i] = floorport.Passage{Source: int64(100 + i%sources), Vector: stored(64, 64, 64, 64)}
	}
	return out
}

func unitEmbedder() *stubEmbedder {
	return &stubEmbedder{model: model, vectors: func(string) []float32 { return []float32{1, 0, 0, 0} }}
}

func TestCalibrateMeasuresTheNoiseAndKeepsTheFloor(t *testing.T) {
	store := newMemoryStore()
	samples := stubSamples{titles: titlesOf(25), passages: passagesOf(60, 3)}
	embedder := unitEmbedder()

	outcome, got, err := floor.NewCalibrate(samples, store, embedder, newClock(calibratedAt)).
		Execute(t.Context(), vault)
	if err != nil || outcome != floor.Calibrated {
		t.Fatalf("got %v, %v", outcome, err)
	}

	// Every pair sits at 0.5, so the median is 0.5 and the spread is the minimum.
	if abs(got.Median-0.5) > 1e-6 || abs(got.Spread-0.01) > 1e-9 || abs(got.Floor-0.52) > 1e-6 {
		t.Errorf("median %v spread %v floor %v, want 0.5, 0.01 and 0.52", got.Median, got.Spread, got.Floor)
	}
	if got.QueryCount != 25 || got.PassageCount != 60 {
		t.Errorf("measured %d queries against %d passages, want 25 against 60", got.QueryCount, got.PassageCount)
	}
	if len(got.Histogram) != 20 || got.Histogram[10] != 25*60 {
		t.Errorf("histogram %v, want %d pairs in bin 10", got.Histogram, 25*60)
	}
	if !got.CalibratedTime.Equal(calibratedAt) {
		t.Errorf("calibrated at %v, want the injected clock's %v", got.CalibratedTime, calibratedAt)
	}

	entry, found, _ := store.ReadFloor(t.Context(), vault, model.Recipe())
	if !found || entry.Calibration == nil || entry.Calibration.Floor != got.Floor {
		t.Errorf("stored %+v, want the calibration under the recipe of the model", entry)
	}
	if len(embedder.embedded) != 1 || len(embedder.embedded[0]) != 25 {
		t.Errorf("titles were embedded in %d calls, want one call of 25", len(embedder.embedded))
	}
}

func TestCalibrateComparesATitleOnlyWithPassagesOfOtherSources(t *testing.T) {
	passages := passagesOf(60, 2)
	// This passage is the title's own note's: identical in direction to every title.
	passages = append(passages, floorport.Passage{Source: 1, Vector: stored(127, 0, 0, 0)})

	_, got, err := floor.NewCalibrate(stubSamples{titles: titlesOf(20), passages: passages},
		newMemoryStore(), unitEmbedder(), newClock(calibratedAt)).Execute(t.Context(), vault)
	if err != nil {
		t.Fatal(err)
	}
	// Titles 2 to 20 meet the passage at similarity 1; title 1 belongs to its source.
	if got.Histogram[19] != 19 {
		t.Errorf("the last bin holds %d pairs, want 19", got.Histogram[19])
	}
	if want := 20*60 + 19; sum(got.Histogram) != want {
		t.Errorf("%d pairs measured, want %d", sum(got.Histogram), want)
	}
}

func TestCalibrateHasTooFewToMeasure(t *testing.T) {
	for _, tc := range []struct {
		name     string
		titles   int
		passages []floorport.Passage
	}{
		{"19 titles", 19, passagesOf(60, 3)},
		{"49 passages", 25, passagesOf(49, 3)},
		{"one source", 25, passagesOf(60, 1)},
		{"nothing", 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newMemoryStore()
			embedder := unitEmbedder()
			outcome, _, err := floor.NewCalibrate(stubSamples{titles: titlesOf(tc.titles), passages: tc.passages},
				store, embedder, newClock(calibratedAt)).Execute(t.Context(), vault)
			if err != nil || outcome != floor.TooFew {
				t.Fatalf("got %v, %v, want TooFew", outcome, err)
			}
			if store.writes != 0 {
				t.Errorf("%d writes, want none", store.writes)
			}
			if len(embedder.embedded) != 0 {
				t.Errorf("the embedder was called for a vault too small to measure")
			}
		})
	}
}

func TestCalibrateThresholdsAreInclusive(t *testing.T) {
	outcome, _, err := floor.NewCalibrate(stubSamples{titles: titlesOf(20), passages: passagesOf(50, 2)},
		newMemoryStore(), unitEmbedder(), newClock(calibratedAt)).Execute(t.Context(), vault)
	if err != nil || outcome != floor.Calibrated {
		t.Errorf("20 titles and 50 passages over 2 sources gave %v, %v, want Done", outcome, err)
	}
}

func TestCalibrateCountsOnlyTitlesThatGotAVector(t *testing.T) {
	embedder := unitEmbedder()
	embedder.vectors = func(text string) []float32 {
		if text == "title 0" {
			return nil
		}
		return []float32{1, 0, 0, 0}
	}
	store := newMemoryStore()
	outcome, _, err := floor.NewCalibrate(stubSamples{titles: titlesOf(20), passages: passagesOf(60, 3)},
		store, embedder, newClock(calibratedAt)).Execute(t.Context(), vault)
	if err != nil || outcome != floor.TooFew || store.writes != 0 {
		t.Errorf("got %v, %v with %d writes, want TooFew and none", outcome, err, store.writes)
	}
}

func TestCalibrateKeepsNothingWhenTheModelChangesDuringTheRun(t *testing.T) {
	changed := embeddingport.EmbeddingModel{Name: "other-model", Dimensions: 4, From: "test"}
	embedder := unitEmbedder()
	embedder.switchAfterEmbed = &changed
	store := newMemoryStore()

	_, _, err := floor.NewCalibrate(stubSamples{titles: titlesOf(25), passages: passagesOf(60, 3)},
		store, embedder, newClock(calibratedAt)).Execute(t.Context(), vault)
	if !errors.Is(err, floor.ErrRecipeChanged) {
		t.Fatalf("got %v, want ErrRecipeChanged", err)
	}
	if store.writes != 0 {
		t.Errorf("%d writes, want none", store.writes)
	}
}

func TestCalibrateStoresForTheRecipeItBeganWith(t *testing.T) {
	store := newMemoryStore()
	_, _, err := floor.NewCalibrate(stubSamples{titles: titlesOf(25), passages: passagesOf(60, 3)},
		store, unitEmbedder(), newClock(calibratedAt)).Execute(t.Context(), vault)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := store.settings[key(vault, model.Recipe())]; !found || len(store.settings) != 1 {
		t.Errorf("stored under %v, want only %s", store.settings, key(vault, model.Recipe()))
	}
}

func TestCalibratePropagatesErrors(t *testing.T) {
	samples := stubSamples{titles: titlesOf(25), passages: passagesOf(60, 3)}
	broken := errors.New("the index is locked")
	for _, tc := range []struct {
		name    string
		run     func(store *memoryStore) error
		want    error
		wantAlt error
	}{
		{
			name: "no embedder",
			run: func(store *memoryStore) error {
				_, _, err := floor.NewCalibrate(samples, store, nil, newClock(calibratedAt)).Execute(t.Context(), vault)
				return err
			},
			want: floor.ErrNoModel,
		},
		{
			name: "a plugin that is down",
			run: func(store *memoryStore) error {
				embedder := unitEmbedder()
				embedder.err = errPluginDown
				_, _, err := floor.NewCalibrate(samples, store, embedder, newClock(calibratedAt)).Execute(t.Context(), vault)
				return err
			},
			want: floor.ErrEmbedding, wantAlt: errPluginDown,
		},
		{
			name: "a sample that cannot be read",
			run: func(store *memoryStore) error {
				_, _, err := floor.NewCalibrate(stubSamples{err: broken}, store, unitEmbedder(), newClock(calibratedAt)).
					Execute(t.Context(), vault)
				return err
			},
			want: broken,
		},
		{
			name: "a stored vector of another width",
			run: func(store *memoryStore) error {
				odd := stubSamples{titles: titlesOf(25), passages: append(passagesOf(60, 3), floorport.Passage{Source: 9, Vector: stored(1, 2)})}
				_, _, err := floor.NewCalibrate(odd, store, unitEmbedder(), newClock(calibratedAt)).Execute(t.Context(), vault)
				return err
			},
			want: errAny,
		},
		{
			name: "a cancelled context",
			run: func(store *memoryStore) error {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				embedder := unitEmbedder()
				_, _, err := floor.NewCalibrate(samples, store, embedder, newClock(calibratedAt)).Execute(ctx, vault)
				return err
			},
			want: context.Canceled,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := newMemoryStore()
			err := tc.run(store)
			if err == nil {
				t.Fatal("no error")
			}
			if tc.want != errAny && !errors.Is(err, tc.want) {
				t.Errorf("got %v, want %v", err, tc.want)
			}
			if tc.wantAlt != nil && !errors.Is(err, tc.wantAlt) {
				t.Errorf("got %v, want it to wrap %v", err, tc.wantAlt)
			}
			if store.writes != 0 {
				t.Errorf("%d writes after a failure, want none", store.writes)
			}
		})
	}
}

var errAny = errors.New("any error")

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func sum(counts []int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}
