package relevance_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/relevance"
)

// calibrate runs one calibration over the search, with the state the host reports.
func calibrate(t *testing.T, search host.Search) (relevance.Outcome, relevance.Calibration, error) {
	t.Helper()
	state, err := search.GetEmbeddingState(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return relevance.NewCalibrator(newClock()).Calibrate(t.Context(), search, state)
}

func TestCalibrateMeasuresTheNoise(t *testing.T) {
	search, _ := newHost(0)
	fill(search, 25, passagesOf(60, 3))

	outcome, got, err := calibrate(t, search)
	if err != nil || outcome != relevance.Calibrated {
		t.Fatalf("got %v, %v", outcome, err)
	}
	// Every pair has similarity 0.5, so the median is 0.5 and the spread is the minimum.
	if abs(got.Median-0.5) > 1e-6 || abs(got.Spread-0.01) > 1e-9 || abs(got.Floor-0.52) > 1e-6 {
		t.Errorf("median %v spread %v floor %v, want 0.5, 0.01 and 0.52", got.Median, got.Spread, got.Floor)
	}
	if got.QueryCount != 25 || got.PassageCount != 60 {
		t.Errorf("measured %d queries against %d passages, want 25 against 60", got.QueryCount, got.PassageCount)
	}
	if len(got.Histogram) != 20 || got.Histogram[10] != 25*60 {
		t.Errorf("histogram %v, want %d pairs in bin 10", got.Histogram, 25*60)
	}
	if !got.CalibratedTime.Equal(measuredAt) {
		t.Errorf("calibrated at %v, want the injected clock's %v", got.CalibratedTime, measuredAt)
	}
	if batches := search.EmbeddedBatches(); len(batches) != 1 || len(batches[0]) != 25 {
		t.Errorf("titles were embedded in %d calls, want one call of 25", len(batches))
	}
}

func TestCalibrateComparesATitleOnlyWithPassagesOfOtherSources(t *testing.T) {
	search, _ := newHost(0)
	// This passage is the title's own note's: identical in direction to every title.
	passages := append(passagesOf(60, 2), host.ChunkVector{Source: 1, Vector: stored(127, 0, 0, 0)})
	fill(search, 20, passages)

	_, got, err := calibrate(t, search)
	if err != nil {
		t.Fatal(err)
	}
	// Titles 2 to 20 meet the passage at similarity 1, and title 1 belongs to its source.
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
		passages []host.ChunkVector
	}{
		{"19 titles", 19, passagesOf(60, 3)},
		{"49 passages", 25, passagesOf(49, 3)},
		{"one source", 25, passagesOf(60, 1)},
		{"nothing", 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			search, _ := newHost(0)
			fill(search, tc.titles, tc.passages)
			outcome, _, err := calibrate(t, search)
			if err != nil || outcome != relevance.TooFew {
				t.Fatalf("got %v, %v, want TooFew", outcome, err)
			}
			if batches := search.EmbeddedBatches(); len(batches) != 0 {
				t.Errorf("the model was called for a vault too small to measure")
			}
		})
	}
}

func TestCalibrateThresholdsAreInclusive(t *testing.T) {
	search, _ := newHost(0)
	fill(search, 20, passagesOf(50, 2))
	outcome, _, err := calibrate(t, search)
	if err != nil || outcome != relevance.Calibrated {
		t.Errorf("20 titles and 50 passages over 2 sources gave %v, %v, want Calibrated", outcome, err)
	}
}

func TestCalibrateCountsOnlyTitlesThatGotAVector(t *testing.T) {
	search, _ := newHost(0)
	search.SetEmbedder(func(text string) []float32 {
		if text == "title 0" {
			return nil
		}
		return []float32{1, 0, 0, 0}
	})
	fill(search, 20, passagesOf(60, 3))
	outcome, _, err := calibrate(t, search)
	if err != nil || outcome != relevance.TooFew {
		t.Errorf("got %v, %v, want TooFew", outcome, err)
	}
}

func TestCalibrateClipsALongTitleAtTheEmbeddingLimit(t *testing.T) {
	search, _ := newHost(0)
	long := strings.Repeat("é", host.MaxEmbeddedBytes)
	titles := titlesOf(25)
	titles[0].Title = long
	search.SetNoteTitles(titles)
	search.SetChunkVectors(modelRecipe, passagesOf(60, 3))
	search.SetProgress(60, 60)

	outcome, _, err := calibrate(t, search)
	if err != nil || outcome != relevance.Calibrated {
		t.Fatalf("got %v, %v", outcome, err)
	}
	clipped := search.EmbeddedBatches()[0][0]
	if len(clipped) > host.MaxEmbeddedBytes || !strings.HasPrefix(long, clipped) || len(clipped) < host.MaxEmbeddedBytes-1 {
		t.Errorf("the title was embedded as %d bytes, want it cut at the limit on a character", len(clipped))
	}
}

// changingSearch is a search whose model answers the titles under another recipe, as a model that
// changed while the titles were embedded does.
type changingSearch struct {
	host.Search
	answer func(embedded host.EmbeddedTexts) (host.EmbeddedTexts, error)
}

func (s changingSearch) EmbedTexts(ctx context.Context, texts []string) (host.EmbeddedTexts, error) {
	embedded, err := s.Search.EmbedTexts(ctx, texts)
	if err != nil {
		return embedded, err
	}
	return s.answer(embedded)
}

func TestCalibrateKeepsNothingWhenTheModelChangesDuringTheRun(t *testing.T) {
	for _, tc := range []struct {
		name   string
		answer func(embedded host.EmbeddedTexts) (host.EmbeddedTexts, error)
	}{
		{"the vectors come under another recipe", func(embedded host.EmbeddedTexts) (host.EmbeddedTexts, error) {
			embedded.Recipe = "other|recipe"
			return embedded, nil
		}},
		{"the host refuses the recipe", func(host.EmbeddedTexts) (host.EmbeddedTexts, error) {
			return host.EmbeddedTexts{}, host.ErrRecipeMismatch
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			search, _ := newHost(0)
			fill(search, 25, passagesOf(60, 3))
			state, err := search.GetEmbeddingState(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			changing := changingSearch{Search: search, answer: tc.answer}

			outcome, got, err := relevance.NewCalibrator(newClock()).Calibrate(t.Context(), changing, state)
			if !errors.Is(err, relevance.ErrRecipeChanged) {
				t.Errorf("got %v, want ErrRecipeChanged", err)
			}
			if outcome != 0 || got.QueryCount != 0 {
				t.Errorf("a calibration of a model that changed returned %v and %+v", outcome, got)
			}
		})
	}
}

func TestCalibrateReportsTheFailuresOfTheHost(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(search *hosttest.Search)
		want error
	}{
		{"a model that fails", func(search *hosttest.Search) { search.FailEmbedding(errors.New("the plugin is not running")) }, host.ErrEmbeddingFailed},
		{"a permission that is denied", func(search *hosttest.Search) { search.Deny(host.PermissionIndexRead) }, host.ErrPermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			search, _ := newHost(0)
			fill(search, 25, passagesOf(60, 3))
			state, err := search.GetEmbeddingState(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			tc.set(search)
			_, _, err = relevance.NewCalibrator(newClock()).Calibrate(t.Context(), search, state)
			if !errors.Is(err, tc.want) {
				t.Errorf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCalibrateRefusesAStoredVectorOfAnotherWidth(t *testing.T) {
	search, _ := newHost(0)
	fill(search, 25, append(passagesOf(60, 3), host.ChunkVector{Source: 9, Vector: stored(1, 2)}))
	if _, _, err := calibrate(t, search); err == nil {
		t.Error("a vector of another width was measured")
	}
}

func TestCalibrateStopsWhenTheContextEnds(t *testing.T) {
	search, _ := newHost(0)
	fill(search, 25, passagesOf(60, 3))
	state, err := search.GetEmbeddingState(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := relevance.NewCalibrator(newClock()).Calibrate(ctx, search, state); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want the cancellation", err)
	}
}
