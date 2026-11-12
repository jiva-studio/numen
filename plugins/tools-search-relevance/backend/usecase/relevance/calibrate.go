// Package relevance decides the similarity a passage has to reach to answer a search by meaning,
// measures it on a vault, and keeps the result in the host.
package relevance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

const (
	// titleSample is how many note titles a calibration takes from the vault at most.
	titleSample = 200

	// minQueries, minPassages and minSources are what a vault has to hold before a measurement is
	// meaningful.
	minQueries  = 20
	minPassages = 50
	minSources  = 2
)

// ErrRecipeChanged is a calibration whose model changed while it ran. The similarities it measured
// belong to no single recipe and are not kept.
var ErrRecipeChanged = errors.New("the embedding model changed during the calibration")

// Outcome is how a calibration ended when it did not fail.
type Outcome int

const (
	// Calibrated: the floor was measured.
	Calibrated Outcome = iota + 1
	// TooFew: the vault holds too few embedded notes to measure, and nothing was measured.
	TooFew
)

// Calibration is what one measurement of a vault under one recipe found.
type Calibration struct {
	Floor          float64
	Median         float64
	Spread         float64
	Histogram      []int
	QueryCount     int
	PassageCount   int
	CalibratedTime time.Time
}

// Clock reads the time.
type Clock interface {
	Now() time.Time
}

// Calibrator measures where the model in use puts text about different things in a vault.
//
// The note titles are the queries. The host compares each with the passages of every other note, and
// those pairs are not about each other: the similarities between them are the noise a floor has to
// stand above.
type Calibrator struct {
	clock Clock
}

// NewCalibrator returns the calibrator that stamps its measurements with the clock.
func NewCalibrator(clock Clock) Calibrator {
	return Calibrator{clock: clock}
}

// Calibrate measures the vault the search and the notes work, under the recipe of the state.
func (c Calibrator) Calibrate(ctx context.Context, index host.SearchIndex, notes host.NoteLookup, state host.EmbeddingState) (Outcome, Calibration, error) {
	titles, err := readTitles(ctx, notes)
	if err != nil {
		return 0, Calibration{}, err
	}
	if len(titles) < minQueries {
		return TooFew, Calibration{}, nil
	}
	counts, err := index.SampleSimilarities(ctx, state.Recipe, nil)
	if err != nil {
		return 0, Calibration{}, fmt.Errorf("sample the passages: %w", err)
	}
	if isTooSmall(counts) {
		return TooFew, Calibration{}, nil
	}

	queries, err := embedTitles(ctx, index, state.Recipe, titles)
	if err != nil {
		return 0, Calibration{}, err
	}
	sample, err := index.SampleSimilarities(ctx, state.Recipe, queries)
	switch {
	case errors.Is(err, host.ErrRecipeMismatch):
		return 0, Calibration{}, fmt.Errorf("%w: %w", ErrRecipeChanged, err)
	case err != nil:
		return 0, Calibration{}, fmt.Errorf("sample the similarities: %w", err)
	case len(sample.Rows) != len(queries):
		return 0, Calibration{}, fmt.Errorf("sample the similarities: %d rows for %d queries", len(sample.Rows), len(queries))
	}
	if isTooSmall(sample) {
		return TooFew, Calibration{}, nil
	}
	similarities, measured := collectNoise(queries, sample)
	if measured < minQueries {
		return TooFew, Calibration{}, nil
	}

	stats := Summarize(similarities)
	return Calibrated, Calibration{
		Floor:          stats.Floor,
		Median:         stats.Median,
		Spread:         stats.Spread,
		Histogram:      stats.Histogram,
		QueryCount:     measured,
		PassageCount:   sample.PassageCount,
		CalibratedTime: c.clock.Now(),
	}, nil
}

func isTooSmall(sample host.SimilaritySample) bool {
	return sample.PassageCount < minPassages || sample.SourceCount < minSources
}

type noteTitle struct {
	path string
	text string
}

func readTitles(ctx context.Context, notes host.NoteLookup) ([]noteTitle, error) {
	paths, err := notes.GetPathsOfType(ctx, domain.TypeNote)
	if err != nil {
		return nil, fmt.Errorf("list the notes: %w", err)
	}
	titles := make([]noteTitle, 0, min(len(paths), titleSample))
	for start := 0; start < len(paths) && len(titles) < titleSample; start += titleSample {
		batch := paths[start:min(start+titleSample, len(paths))]
		refs, err := notes.GetNotes(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("read the note titles: %w", err)
		}
		for _, path := range batch {
			if ref, found := refs[path]; found && strings.TrimSpace(ref.Title) != "" && len(titles) < titleSample {
				titles = append(titles, noteTitle{path: path, text: ref.Title})
			}
		}
	}
	return titles, nil
}

func embedTitles(ctx context.Context, index host.SearchIndex, recipe string, titles []noteTitle) ([]host.SimilarityQuery, error) {
	texts := make([]string, len(titles))
	for i, title := range titles {
		texts[i] = clipText(title.text, host.MaxEmbeddedBytes)
	}
	embedded, err := index.EmbedTexts(ctx, texts)
	switch {
	case errors.Is(err, host.ErrRecipeMismatch):
		return nil, fmt.Errorf("%w: %w", ErrRecipeChanged, err)
	case err != nil:
		return nil, fmt.Errorf("embed the titles: %w", err)
	case embedded.Recipe != recipe:
		return nil, ErrRecipeChanged
	case len(embedded.Vectors) != len(titles):
		return nil, fmt.Errorf("embed the titles: %w: %d vectors for %d titles", host.ErrEmbeddingFailed, len(embedded.Vectors), len(titles))
	}
	queries := make([]host.SimilarityQuery, len(titles))
	for i, title := range titles {
		queries[i] = host.SimilarityQuery{Path: title.path, Vector: embedded.Vectors[i]}
	}
	return queries, nil
}

func clipText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	end := limit
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end]
}

func collectNoise(queries []host.SimilarityQuery, sample host.SimilaritySample) ([]float64, int) {
	// gomutants:disable-next-line ARITHMETIC_BASE reason="Slice capacity hint does not affect collected values."
	similarities := make([]float64, 0, len(queries)*sample.PassageCount)
	measured := 0
	for i, query := range queries {
		if len(query.Vector) == 0 {
			continue
		}
		measured++
		similarities = append(similarities, sample.Rows[i]...)
	}
	return similarities, measured
}
