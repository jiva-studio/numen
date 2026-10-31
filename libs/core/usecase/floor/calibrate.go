package floor

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/internal/vector"
	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

const (
	// titleSample and passageSample are how many titles and passages a
	// calibration reads from the vault at most.
	titleSample   = 200
	passageSample = 500

	// minQueries, minPassages and minSources are what a vault has to hold before a
	// measurement is meaningful.
	minQueries  = 20
	minPassages = 50
	minSources  = 2
)

// ErrNoModel is returned by a floor request where the installation has no embedding model.
var ErrNoModel = errors.New("no embedding model")

// ErrEmbedding is an embedding model that could not answer: a plugin that is
// missing or unreachable, or a service that rejected the request.
var ErrEmbedding = errors.New("the embedding model could not answer")

// ErrRecipeChanged is a calibration whose model changed while it ran. The
// similarities it measured belong to no single recipe and are not kept.
var ErrRecipeChanged = errors.New("the embedding model changed during the calibration")

// Outcome is how a calibration ended when it did not fail.
type Outcome int

const (
	// Calibrated: the floor was measured and stored.
	Calibrated Outcome = iota + 1
	// TooFew: the vault holds too few embedded notes to measure, and nothing was stored.
	TooFew
)

// Calibrate measures where the model in use puts text about different things in
// a vault, and keeps the floor that follows.
//
// The note titles are the queries. Each is compared with the passages of every
// other note; those pairs are not about each other, and the similarities between
// them are the noise a floor has to stand above.
type Calibrate struct {
	samples  floorport.Samples
	store    floorport.Store
	embedder embeddingport.Embedder
	clock    machineport.Clock
}

// NewCalibrate returns the calibration over the embedder given. A nil embedder
// has no model, and every calibration then fails with ErrNoModel.
func NewCalibrate(samples floorport.Samples, store floorport.Store, embedder embeddingport.Embedder, clock machineport.Clock) Calibrate {
	return Calibrate{samples: samples, store: store, embedder: embedder, clock: clock}
}

// Execute measures the vault and stores the calibration under the recipe the
// model had when it began.
func (u Calibrate) Execute(ctx context.Context, v domain.Vault) (Outcome, floorport.Calibration, error) {
	if u.embedder == nil {
		return 0, floorport.Calibration{}, ErrNoModel
	}
	recipe := u.embedder.GetModel().Recipe()

	titles, passages, err := u.readSamples(ctx, v, recipe)
	if err != nil {
		return 0, floorport.Calibration{}, err
	}
	if len(titles) < minQueries || len(passages) < minPassages || countSources(passages) < minSources {
		return TooFew, floorport.Calibration{}, nil
	}

	vectors, err := u.embedTitles(ctx, v, titles)
	if err != nil {
		return 0, floorport.Calibration{}, err
	}
	if u.embedder.GetModel().Recipe() != recipe {
		return 0, floorport.Calibration{}, ErrRecipeChanged
	}

	similarities, queries, err := measureNoise(ctx, titles, vectors, passages)
	if err != nil {
		return 0, floorport.Calibration{}, err
	}
	if queries < minQueries {
		return TooFew, floorport.Calibration{}, nil
	}

	stats := Summarize(similarities)
	calibration := floorport.Calibration{
		Floor:          stats.Floor,
		Median:         stats.Median,
		Spread:         stats.Spread,
		Histogram:      stats.Histogram,
		QueryCount:     queries,
		PassageCount:   len(passages),
		CalibratedTime: u.clock.Now(),
	}
	if err := u.store.WriteCalibration(ctx, v, recipe, calibration); err != nil {
		return 0, floorport.Calibration{}, err
	}
	return Calibrated, calibration, nil
}

// readSamples reads the titles and the passages a calibration measures.
func (u Calibrate) readSamples(ctx context.Context, v domain.Vault, recipe string) ([]floorport.Title, []floorport.Passage, error) {
	titles, err := u.samples.GetTitles(ctx, v, titleSample)
	if err != nil {
		return nil, nil, err
	}
	passages, err := u.samples.GetPassages(ctx, v, recipe, passageSample)
	if err != nil {
		return nil, nil, err
	}
	return titles, passages, nil
}

// embedTitles returns one vector per title, in one call to the embedder.
func (u Calibrate) embedTitles(ctx context.Context, v domain.Vault, titles []floorport.Title) ([][]float32, error) {
	texts := make([]string, len(titles))
	for i, title := range titles {
		texts[i] = title.Text
	}
	vectors, err := u.embedder.Embed(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("%w: embed the titles of %s: %w", ErrEmbedding, v.ID, err)
	}
	if len(vectors) != len(titles) {
		return nil, fmt.Errorf("%w: embed the titles of %s: %d vectors for %d titles", ErrEmbedding, v.ID, len(vectors), len(titles))
	}
	return vectors, nil
}

// measureNoise is the similarity of each title with each passage of another
// source, and how many titles were measured. A title without a vector is not
// measured.
func measureNoise(ctx context.Context, titles []floorport.Title, vectors [][]float32, passages []floorport.Passage) ([]float64, int, error) {
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
			similarities = append(similarities, vector.SimilarityBytes(query, passage.Vector))
		}
	}
	return similarities, queries, nil
}

// countSources is how many sources the passages come from.
func countSources(passages []floorport.Passage) int {
	members := make(map[int64]struct{}, len(passages))
	for _, passage := range passages {
		members[passage.Source] = struct{}{}
	}
	return len(members)
}
