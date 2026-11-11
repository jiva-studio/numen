package search

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchpolicyport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Progress reads how far a vault is embedded.
type Progress interface {
	// GetProgress returns how many of the vault's chunks can have a vector, and how many of those
	// have one for the named model.
	GetProgress(ctx context.Context, vaultID domain.VaultID, model string) (held, embedded int64, err error)
}

// ErrInvalidFloor is a floor outside the range above zero and up to one.
var ErrInvalidFloor = errors.New("a floor is a cosine similarity above 0 and at most 1")

// FloorSourceKind is where the effective floor of a vault comes from.
type FloorSourceKind int

const (
	// FloorFromDefault: no other source answered.
	FloorFromDefault FloorSourceKind = iota + 1
	// FloorFromModel: the starting floor the plugin of the embedding model reports.
	FloorFromModel
	// FloorFromStore: a plugin stored the floor for the vault and the recipe.
	FloorFromStore
)

// EffectiveFloor is the floor a search applies to a vault under a recipe, and the floor of each
// source it was chosen from. A source with no floor holds zero.
type EffectiveFloor struct {
	Floor   float64
	Source  FloorSourceKind
	Stored  float64
	Hint    float64
	Default float64
}

// FloorHint returns the starting floor of the model in use, and zero where it has none.
type FloorHint func(ctx context.Context) float64

// FloorPolicy decides the floor that a search by meaning applies when its query
// names none. The first source with an answer wins: the floor stored for the
// vault under the recipe, then the hint of the embedding model, then DefaultFloor.
type FloorPolicy struct {
	store        searchpolicyport.FloorStore
	hint         FloorHint
	errorHandler func(error)
}

// NewFloorPolicy returns the policy over a store of floors. A nil store holds no floors, a nil hint
// reports none, and a nil errorHandler ignores the errors of the store.
func NewFloorPolicy(store searchpolicyport.FloorStore, hint FloorHint, errorHandler func(error)) FloorPolicy {
	if errorHandler == nil {
		errorHandler = func(error) {}
	}
	return FloorPolicy{store: store, hint: hint, errorHandler: errorHandler}
}

// Get returns the floor in force for the vault under the recipe. A store that cannot be read is
// reported to the error handler and leaves the floor to the sources after it.
func (p FloorPolicy) Get(ctx context.Context, v domain.Vault, recipe string) EffectiveFloor {
	out := EffectiveFloor{Default: DefaultFloor}
	if p.store != nil && recipe != "" {
		stored, found, err := p.store.ReadFloor(ctx, v, recipe)
		switch {
		case err != nil:
			p.errorHandler(err)
		case found && isValidFloor(stored):
			out.Stored = stored
		}
	}
	if p.hint != nil {
		if hint := p.hint(ctx); isValidFloor(hint) {
			out.Hint = hint
		}
	}
	switch {
	case out.Stored > 0:
		out.Floor, out.Source = out.Stored, FloorFromStore
	case out.Hint > 0:
		out.Floor, out.Source = out.Hint, FloorFromModel
	default:
		out.Floor, out.Source = DefaultFloor, FloorFromDefault
	}
	return out
}

// GetFloor returns the floor in force for the vault under the recipe.
func (p FloorPolicy) GetFloor(ctx context.Context, v domain.Vault, recipe string) float64 {
	return p.Get(ctx, v, recipe).Floor
}

// GetEmbeddingState reads the model in use and the progress of the vault under it from the index.
func (p FloorPolicy) GetEmbeddingState(
	ctx context.Context, v domain.Vault, progress Progress, model embedding.Model,
) (plugin.EmbeddingState, error) {
	recipe := model.Recipe()
	held, embedded, err := progress.GetProgress(ctx, v.ID, recipe)
	if err != nil {
		return plugin.EmbeddingState{}, fmt.Errorf("read the embedding progress of %s: %w", v.ID, err)
	}
	return plugin.EmbeddingState{
		Recipe:        recipe,
		Model:         model.Name,
		FloorHint:     p.Get(ctx, v, recipe).Hint,
		ChunkCount:    held,
		EmbeddedCount: embedded,
	}, nil
}

// Set stores the floor of the vault under the recipe, and removes the stored floor when the floor is
// zero.
func (p FloorPolicy) Set(ctx context.Context, v domain.Vault, recipe string, floor float64) error {
	if floor != 0 && !isValidFloor(floor) {
		return ErrInvalidFloor
	}
	if p.store == nil {
		return nil
	}
	if floor == 0 {
		return p.store.DeleteFloor(ctx, v, recipe)
	}
	return p.store.WriteFloor(ctx, v, recipe, floor)
}

func isValidFloor(floor float64) bool {
	return floor > 0 && floor <= 1
}
