// Package relevance decides the similarity a passage has to reach to answer a search by meaning,
// measures it on a vault, and keeps the result in the host.
package relevance

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

const (
	// titleSample and passageSample are how many titles and passages a calibration reads from the
	// vault at most.
	titleSample   = 200
	passageSample = 500

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
	// Floor is the similarity a passage has to reach, as the measurement derives it.
	Floor float64
	// Median and Spread describe the similarities between queries and passages that are not about them.
	Median float64
	Spread float64
	// Histogram counts those similarities in equal bins from zero to one.
	Histogram []int
	// QueryCount and PassageCount are how many queries were measured against how many passages.
	QueryCount   int
	PassageCount int
	// CalibratedTime is when the measurement ended.
	CalibratedTime time.Time
}

// Clock reads the time.
type Clock interface {
	Now() time.Time
}

// Calibrator measures where the model in use puts text about different things in a vault.
//
// The note titles are the queries. Each is compared with the passages of every other note, and those
// pairs are not about each other: the similarities between them are the noise a floor has to stand above.
type Calibrator struct {
	clock Clock
}

// NewCalibrator returns the calibrator that stamps its measurements with the clock.
func NewCalibrator(clock Clock) Calibrator {
	return Calibrator{clock: clock}
}

// Calibrate measures the vault the search works under the recipe of the state.
func (c Calibrator) Calibrate(ctx context.Context, index host.Search, state host.EmbeddingState) (Outcome, Calibration, error) {
	titles, passages, err := readSamples(ctx, index, state.Recipe)
	if err != nil {
		return 0, Calibration{}, err
	}
	if len(titles) < minQueries || len(passages) < minPassages || countSources(passages) < minSources {
		return TooFew, Calibration{}, nil
	}

	vectors, err := embedTitles(ctx, index, state.Recipe, titles)
	if err != nil {
		return 0, Calibration{}, err
	}
	similarities, queries, err := measureNoise(ctx, titles, vectors, passages)
	if err != nil {
		return 0, Calibration{}, err
	}
	if queries < minQueries {
		return TooFew, Calibration{}, nil
	}

	stats := Summarize(similarities)
	return Calibrated, Calibration{
		Floor:          stats.Floor,
		Median:         stats.Median,
		Spread:         stats.Spread,
		Histogram:      stats.Histogram,
		QueryCount:     queries,
		PassageCount:   len(passages),
		CalibratedTime: c.clock.Now(),
	}, nil
}

// readSamples reads the titles and the passages a calibration measures.
func readSamples(ctx context.Context, index host.Search, recipe string) ([]host.NoteTitle, []host.ChunkVector, error) {
	titles, err := index.ListNoteTitles(ctx, titleSample)
	if err != nil {
		return nil, nil, fmt.Errorf("read the note titles: %w", err)
	}
	passages, err := index.ListSmallChunkVectors(ctx, recipe, passageSample)
	if err != nil {
		return nil, nil, fmt.Errorf("read the stored vectors: %w", err)
	}
	return titles, passages, nil
}

// embedTitles returns one vector per title, in one call to the model.
func embedTitles(ctx context.Context, index host.Search, recipe string, titles []host.NoteTitle) ([][]float32, error) {
	texts := make([]string, len(titles))
	for i, title := range titles {
		texts[i] = clipText(title.Title, host.MaxEmbeddedBytes)
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
	return embedded.Vectors, nil
}

// clipText is the text cut to at most limit bytes at the end of a character.
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

// measureNoise is the similarity of each title with each passage of another source, and how many
// titles were measured. A title without a vector is not measured.
func measureNoise(ctx context.Context, titles []host.NoteTitle, vectors [][]float32, passages []host.ChunkVector) ([]float64, int, error) {
	similarities := make([]float64, 0, len(titles)*len(passages))
	queries := 0
	for i, title := range titles {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		query := vectors[i]
		if len(query) == 0 {
			continue
		}
		queries++
		for _, passage := range passages {
			if passage.Source == title.Source {
				continue
			}
			if len(passage.Vector) != len(query) {
				return nil, 0, fmt.Errorf("a passage holds %d dimensions where the title has %d", len(passage.Vector), len(query))
			}
			similarities = append(similarities, getSimilarity(query, passage.Vector))
		}
	}
	return similarities, queries, nil
}

// countSources is how many sources the passages come from.
func countSources(passages []host.ChunkVector) int {
	members := make(map[int64]struct{}, len(passages))
	for _, passage := range passages {
		members[passage.Source] = struct{}{}
	}
	return len(members)
}
