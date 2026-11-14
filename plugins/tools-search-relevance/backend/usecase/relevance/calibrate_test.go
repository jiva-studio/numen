package relevance_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/relevance"
)

func calibrate(t *testing.T, search host.SearchIndex, vault host.NoteLookup) (relevance.Outcome, relevance.Calibration, error) {
	t.Helper()
	state, err := search.GetEmbeddingState(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return relevance.NewCalibrator(newClock()).Calibrate(t.Context(), search, vault, state)
}

func TestCalibrateMeasuresTheNoise(t *testing.T) {
	if relevance.Calibrated != 1 || relevance.TooFew != 2 {
		t.Fatalf("outcomes %d and %d, want 1 and 2", relevance.Calibrated, relevance.TooFew)
	}
	search, vault := newHost(0)
	fill(search, vault, 25, sampledChunksOf(60, 3))

	outcome, got, err := calibrate(t, search, vault)
	if err != nil || outcome != relevance.Calibrated {
		t.Fatalf("got %v, %v", outcome, err)
	}
	// Every pair has similarity 0.5, so the median is 0.5 and the spread is the minimum.
	if abs(got.Median-0.5) > 1e-6 || abs(got.Spread-0.01) > 1e-9 || abs(got.Floor-0.52) > 1e-6 {
		t.Errorf("median %v spread %v floor %v, want 0.5, 0.01 and 0.52", got.Median, got.Spread, got.Floor)
	}
	if got.QueryCount != 25 || got.ChunkCount != 60 {
		t.Errorf("measured %d queries against %d passages, want 25 against 60", got.QueryCount, got.ChunkCount)
	}
	if len(got.Histogram) != 20 || got.Histogram[10] != 25*60 {
		t.Errorf("histogram %v, want %d pairs in bin 10", got.Histogram, 25*60)
	}
	if !got.CalibratedTime.Equal(getMeasuredAt()) {
		t.Errorf("calibrated at %v, want the injected clock's %v", got.CalibratedTime, getMeasuredAt())
	}
	if batches := search.EmbeddedBatches(); len(batches) != 1 || len(batches[0]) != 25 {
		t.Errorf("titles were embedded in %d calls, want one call of 25", len(batches))
	}
}

func TestCalibrateComparesATitleOnlyWithChunksOfOtherSources(t *testing.T) {
	search, vault := newHost(0)
	// This passage is the first note's: identical in direction to every title.
	passages := append(sampledChunksOf(60, 2), hosttest.SampledChunk{Path: notePath(1), Vector: []float32{127, 0, 0, 0}})
	fill(search, vault, 20, passages)

	_, got, err := calibrate(t, search, vault)
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
		passages []hosttest.SampledChunk
	}{
		{"19 titles", 19, sampledChunksOf(60, 3)},
		{"49 passages", 25, sampledChunksOf(49, 3)},
		{"one source", 25, sampledChunksOf(60, 1)},
		{"nothing", 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			search, vault := newHost(0)
			fill(search, vault, tc.titles, tc.passages)
			outcome, _, err := calibrate(t, search, vault)
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
	search, vault := newHost(0)
	fill(search, vault, 20, sampledChunksOf(50, 2))
	outcome, _, err := calibrate(t, search, vault)
	if err != nil || outcome != relevance.Calibrated {
		t.Errorf("20 titles and 50 passages over 2 sources gave %v, %v, want Calibrated", outcome, err)
	}
}

func TestCalibrateCountsOnlyTitlesThatGotAVector(t *testing.T) {
	search, vault := newHost(0)
	search.SetEmbedder(func(text string) []float32 {
		if text == "title 0" {
			return nil
		}
		return []float32{1, 0, 0, 0}
	})
	fill(search, vault, 20, sampledChunksOf(60, 3))
	outcome, _, err := calibrate(t, search, vault)
	if err != nil || outcome != relevance.TooFew {
		t.Errorf("got %v, %v, want TooFew", outcome, err)
	}

	fill(search, vault, 21, sampledChunksOf(60, 3))
	outcome, got, err := calibrate(t, search, vault)
	if err != nil || outcome != relevance.Calibrated || got.QueryCount != 20 {
		t.Errorf("21 titles with one empty vector gave %v, %+v, %v, want Calibrated with 20 queries", outcome, got, err)
	}
}

type noteLookup struct {
	host.NoteLookup
	paths    []string
	pathsErr error
	getNotes func(ctx context.Context, paths []string) (map[string]domain.NoteRef, error)
}

func (n noteLookup) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	if n.pathsErr != nil {
		return nil, n.pathsErr
	}
	if n.paths != nil {
		return n.paths, nil
	}
	return n.NoteLookup.GetPathsOfType(ctx, noteType)
}

func (n noteLookup) GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error) {
	if n.getNotes != nil {
		return n.getNotes(ctx, paths)
	}
	return n.NoteLookup.GetNotes(ctx, paths)
}

func TestCalibrateSamplesAtMostTwoHundredNonBlankTitles(t *testing.T) {
	search, vault := newHost(0)
	fill(search, vault, 25, sampledChunksOf(60, 3))

	paths := make([]string, 420)
	for i := range paths {
		paths[i] = fmt.Sprintf("note-%03d.md", i+1)
	}
	lookup := noteLookup{
		NoteLookup: vault,
		paths:      paths,
		getNotes: func(_ context.Context, batch []string) (map[string]domain.NoteRef, error) {
			if len(batch) == 0 {
				return nil, errors.New("empty batch")
			}
			refs := make(map[string]domain.NoteRef, len(batch))
			for _, p := range batch {
				var index int
				if _, err := fmt.Sscanf(p, "note-%03d.md", &index); err != nil {
					return nil, err
				}
				title := "   "
				if index > 205 {
					title = fmt.Sprintf("title %d", index)
				}
				refs[p] = domain.NoteRef{Path: p, Title: title}
			}
			return refs, nil
		},
	}

	outcome, got, err := calibrate(t, search, lookup)
	if err != nil || outcome != relevance.Calibrated || got.QueryCount != 200 {
		t.Fatalf("got %v, %+v, %v, want 200 queries", outcome, got, err)
	}
	batch := search.EmbeddedBatches()[0]
	if batch[0] != "title 206" || batch[199] != "title 405" {
		t.Errorf("embedded %q..%q, want title 206..title 405", batch[0], batch[199])
	}

	exactBatchLookup := noteLookup{
		NoteLookup: vault,
		paths:      paths[:200],
		getNotes: func(_ context.Context, batch []string) (map[string]domain.NoteRef, error) {
			if len(batch) == 0 {
				return nil, errors.New("empty batch")
			}
			refs := make(map[string]domain.NoteRef, len(batch))
			for i, p := range batch {
				if i < 20 {
					refs[p] = domain.NoteRef{Path: p, Title: fmt.Sprintf("title %d", i)}
				}
			}
			return refs, nil
		},
	}
	if outcome, _, err := calibrate(t, search, exactBatchLookup); err != nil || outcome != relevance.Calibrated {
		t.Errorf("200 paths with 20 titles gave %v, %v", outcome, err)
	}

	calls := 0
	fullFirstBatchLookup := noteLookup{
		NoteLookup: vault,
		paths:      paths[:210],
		getNotes: func(_ context.Context, batch []string) (map[string]domain.NoteRef, error) {
			calls++
			if calls > 1 {
				return nil, errors.New("extra batch after reaching sample limit")
			}
			refs := make(map[string]domain.NoteRef, len(batch))
			for i, p := range batch {
				refs[p] = domain.NoteRef{Path: p, Title: fmt.Sprintf("title %d", i)}
			}
			return refs, nil
		},
	}
	if outcome, _, err := calibrate(t, search, fullFirstBatchLookup); err != nil || outcome != relevance.Calibrated {
		t.Errorf("210 titled paths gave %v, %v", outcome, err)
	}
}

func TestCalibrateClipsALongTitleAtTheEmbeddingLimit(t *testing.T) {
	search, vault := newHost(0)
	exact := strings.Repeat("a", host.MaxEmbeddedBytes)
	multiByte := strings.Repeat("€", host.MaxEmbeddedBytes)
	continuation := strings.Repeat("\x80", host.MaxEmbeddedBytes+5)
	fill(search, vault, 25, sampledChunksOf(60, 3))
	lookup := noteLookup{
		NoteLookup: vault,
		getNotes: func(ctx context.Context, batch []string) (map[string]domain.NoteRef, error) {
			refs, err := vault.GetNotes(ctx, batch)
			if err != nil {
				return nil, err
			}
			refs[batch[0]] = domain.NoteRef{Path: batch[0], Title: exact}
			refs[batch[1]] = domain.NoteRef{Path: batch[1], Title: multiByte}
			refs[batch[2]] = domain.NoteRef{Path: batch[2], Title: continuation}
			return refs, nil
		},
	}

	outcome, _, err := calibrate(t, search, lookup)
	if err != nil || outcome != relevance.Calibrated {
		t.Fatalf("got %v, %v", outcome, err)
	}
	embedded := search.EmbeddedBatches()[0]
	if embedded[0] != exact {
		t.Errorf("exact limit title was embedded as %d bytes, want %d", len(embedded[0]), len(exact))
	}
	if len(embedded[1]) != host.MaxEmbeddedBytes-(host.MaxEmbeddedBytes%3) || !utf8.ValidString(embedded[1]) {
		t.Errorf("multibyte title was embedded as %d bytes (valid=%v), want %d valid bytes",
			len(embedded[1]), utf8.ValidString(embedded[1]), host.MaxEmbeddedBytes-(host.MaxEmbeddedBytes%3))
	}
	if embedded[2] != "" {
		t.Errorf("continuation-only title was embedded as %q, want empty", embedded[2])
	}
}

// changingSearch is a search whose model or index changes between calls.
type changingSearch struct {
	host.SearchIndex
	answer func(embedded host.EmbeddedTexts) (host.EmbeddedTexts, error)
	sample func(ctx context.Context, recipe string, queries []host.SimilarityQuery) (host.SimilaritySample, error)
}

func (s changingSearch) EmbedTexts(ctx context.Context, texts []string) (host.EmbeddedTexts, error) {
	embedded, err := s.SearchIndex.EmbedTexts(ctx, texts)
	if err != nil {
		return embedded, err
	}
	if s.answer != nil {
		return s.answer(embedded)
	}
	return embedded, nil
}

func (s changingSearch) SampleSimilarities(ctx context.Context, recipe string, queries []host.SimilarityQuery) (host.SimilaritySample, error) {
	if s.sample != nil && queries != nil {
		return s.sample(ctx, recipe, queries)
	}
	return s.SearchIndex.SampleSimilarities(ctx, recipe, queries)
}

func TestCalibrateKeepsNothingWhenTheModelChangesDuringTheRun(t *testing.T) {
	for _, tc := range []struct {
		name         string
		answer       func(embedded host.EmbeddedTexts) (host.EmbeddedTexts, error)
		sample       func(ctx context.Context, recipe string, queries []host.SimilarityQuery) (host.SimilaritySample, error)
		wantMismatch bool
	}{
		{
			name: "the vectors come under another recipe",
			answer: func(embedded host.EmbeddedTexts) (host.EmbeddedTexts, error) {
				embedded.Recipe = "other|recipe"
				return embedded, nil
			},
		},
		{
			name: "the host refuses the recipe during embedding",
			answer: func(host.EmbeddedTexts) (host.EmbeddedTexts, error) {
				return host.EmbeddedTexts{}, host.ErrRecipeMismatch
			},
			wantMismatch: true,
		},
		{
			name: "the host refuses the recipe during sampling",
			sample: func(context.Context, string, []host.SimilarityQuery) (host.SimilaritySample, error) {
				return host.SimilaritySample{}, host.ErrRecipeMismatch
			},
			wantMismatch: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			search, vault := newHost(0)
			fill(search, vault, 25, sampledChunksOf(60, 3))
			state, err := search.GetEmbeddingState(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			changing := changingSearch{SearchIndex: search, answer: tc.answer, sample: tc.sample}

			outcome, got, err := relevance.NewCalibrator(newClock()).Calibrate(t.Context(), changing, vault, state)
			if !errors.Is(err, relevance.ErrRecipeChanged) {
				t.Errorf("got %v, want ErrRecipeChanged", err)
			}
			if tc.wantMismatch && !errors.Is(err, host.ErrRecipeMismatch) {
				t.Errorf("got %v, want wrapped ErrRecipeMismatch", err)
			}
			if outcome != 0 || got.QueryCount != 0 {
				t.Errorf("a calibration of a model that changed returned %v and %+v", outcome, got)
			}
		})
	}

	search, vault := newHost(0)
	fill(search, vault, 25, sampledChunksOf(60, 3))
	state, err := search.GetEmbeddingState(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	shrinking := changingSearch{
		SearchIndex: search,
		sample: func(_ context.Context, _ string, queries []host.SimilarityQuery) (host.SimilaritySample, error) {
			return host.SimilaritySample{ChunkCount: 10, SourceCount: 3, Rows: make([][]float64, len(queries))}, nil
		},
	}
	if outcome, _, err := relevance.NewCalibrator(newClock()).Calibrate(t.Context(), shrinking, vault, state); err != nil || outcome != relevance.TooFew {
		t.Errorf("shrunk sample gave %v, %v, want TooFew", outcome, err)
	}
}

func TestCalibrateReportsTheFailuresOfTheHost(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(search *hosttest.Search, vault *hosttest.Vault) (host.SearchIndex, host.NoteLookup)
		want error
	}{
		{
			name: "a model that fails",
			set: func(search *hosttest.Search, vault *hosttest.Vault) (host.SearchIndex, host.NoteLookup) {
				search.FailEmbedding(errors.New("the plugin is not running"))
				return search, vault
			},
			want: host.ErrEmbeddingFailed,
		},
		{
			name: "a permission that is denied",
			set: func(search *hosttest.Search, vault *hosttest.Vault) (host.SearchIndex, host.NoteLookup) {
				search.Deny(host.PermissionIndexRead)
				return search, vault
			},
			want: host.ErrPermissionDenied,
		},
		{
			name: "listing notes fails",
			set: func(search *hosttest.Search, vault *hosttest.Vault) (host.SearchIndex, host.NoteLookup) {
				return search, noteLookup{NoteLookup: vault, pathsErr: host.ErrUnknownVault}
			},
			want: host.ErrUnknownVault,
		},
		{
			name: "reading note titles fails",
			set: func(search *hosttest.Search, vault *hosttest.Vault) (host.SearchIndex, host.NoteLookup) {
				return search, noteLookup{
					NoteLookup: vault,
					getNotes: func(context.Context, []string) (map[string]domain.NoteRef, error) {
						return nil, host.ErrPermissionDenied
					},
				}
			},
			want: host.ErrPermissionDenied,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			search, vault := newHost(0)
			fill(search, vault, 25, sampledChunksOf(60, 3))
			state, err := search.GetEmbeddingState(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			idx, notes := tc.set(search, vault)
			outcome, _, err := relevance.NewCalibrator(newClock()).Calibrate(t.Context(), idx, notes, state)
			if outcome != 0 || !errors.Is(err, tc.want) {
				t.Errorf("got outcome %v, err %v, want 0 and %v", outcome, err, tc.want)
			}
		})
	}
}

func TestCalibrateRefusesAChunkOfAnotherWidth(t *testing.T) {
	search, vault := newHost(0)
	fill(search, vault, 25, append(sampledChunksOf(60, 3), hosttest.SampledChunk{Path: "narrow.md", Vector: []float32{1, 2}}))
	if outcome, _, err := calibrate(t, search, vault); outcome != 0 || !errors.Is(err, host.ErrInvalidQuery) {
		t.Errorf("got outcome %v, err %v, want 0 and ErrInvalidQuery", outcome, err)
	}
}

func TestCalibrateStopsWhenTheContextEnds(t *testing.T) {
	search, vault := newHost(0)
	fill(search, vault, 25, sampledChunksOf(60, 3))
	state, err := search.GetEmbeddingState(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := relevance.NewCalibrator(newClock()).Calibrate(ctx, search, vault, state); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want the cancellation", err)
	}
}
