package floor

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrInvalidFloor is a manual floor outside the range above zero and up to one.
var ErrInvalidFloor = errors.New("a floor is a cosine similarity above 0 and at most 1")

// Set chooses between the calibrated floor and one the person sets, for the model
// in use.
type Set struct {
	store    floorport.Store
	embedder embeddingport.Embedder
}

// NewSet returns the choice over the embedder given. A nil embedder has no model,
// and every choice then fails with ErrNoModel.
func NewSet(store floorport.Store, embedder embeddingport.Embedder) Set {
	return Set{store: store, embedder: embedder}
}

// Execute puts the manual floor in force, or clears it. A cleared floor is kept
// as zero.
func (u Set) Execute(ctx context.Context, v domain.Vault, isManual bool, floor float64) error {
	if isManual && !(floor > 0 && floor <= 1) {
		return ErrInvalidFloor
	}
	if u.embedder == nil {
		return ErrNoModel
	}
	if !isManual {
		floor = 0
	}
	return u.store.WriteManualFloor(ctx, v, u.embedder.GetModel().Recipe(), isManual, floor)
}
