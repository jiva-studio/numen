package floorport

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Calibration is what one measurement of a vault under one recipe found.
type Calibration struct {
	// Floor is the similarity a passage has to reach, as the measurement derives it.
	Floor float64
	// Median and Spread describe the similarities between queries and passages
	// that are not about them.
	Median float64
	Spread float64
	// Histogram counts those similarities in equal bins from zero to one.
	Histogram []int
	// QueryCount and PassageCount are how many queries were measured against how
	// many passages.
	QueryCount   int
	PassageCount int
	// CalibratedTime is when the measurement ended.
	CalibratedTime time.Time
}

// Setting is the floor state kept for one vault under one recipe.
type Setting struct {
	// IsManual is whether the person set the floor, and ManualFloor the value
	// they set.
	IsManual    bool
	ManualFloor float64
	// Calibration is nil before the first measurement.
	Calibration *Calibration
}

// Store keeps the floor of each vault under each recipe.
type Store interface {
	// ReadFloor returns the setting kept for the vault under the recipe, and false
	// where nothing is kept.
	ReadFloor(ctx context.Context, v domain.Vault, recipe string) (Setting, bool, error)
	// WriteCalibration keeps a measurement and leaves the manual floor as it is.
	WriteCalibration(ctx context.Context, v domain.Vault, recipe string, c Calibration) error
	// WriteManualFloor sets or clears the manual floor and leaves the measurement
	// as it is.
	WriteManualFloor(ctx context.Context, v domain.Vault, recipe string, isManual bool, floor float64) error
}

// Title is the name of a note and the source it belongs to.
type Title struct {
	Source int64
	Text   string
}

// Passage is a small chunk that carries a vector, and the source it belongs to.
// Vector is the stored form of the vector.
type Passage struct {
	Source int64
	Vector []byte
}

// Samples reads the texts and vectors a calibration measures.
//
// The order of both is a function of the content hashes in the index, so the
// same index yields the same samples.
type Samples interface {
	// GetTitles returns at most `limit` note titles of the vault.
	GetTitles(ctx context.Context, v domain.Vault, limit int) ([]Title, error)
	// GetPassages returns at most `limit` small chunks of the vault that carry a
	// vector under the recipe.
	GetPassages(ctx context.Context, v domain.Vault, recipe string, limit int) ([]Passage, error)
}

// Nearest reads how near the best passage of a vault stands to a vector.
type Nearest interface {
	// GetBestSimilarity returns the cosine similarity of the passage of the vault
	// nearest the query vector under the recipe, and zero where there is none.
	GetBestSimilarity(ctx context.Context, vaultID domain.VaultID, recipe string, query []float32) (float64, error)
}
