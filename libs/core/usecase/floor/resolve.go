package floor

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Source is where the floor in force comes from.
type Source int

const (
	// SourceDefault: no other source answered.
	SourceDefault Source = iota
	// SourceFile: the settings file names one floor for every model.
	SourceFile
	// SourceManual: the person set the floor for the model in use.
	SourceManual
	// SourceCalibrated: measured on the vault with the model in use.
	SourceCalibrated
	// SourceModel: the starting floor the model's plugin reports.
	SourceModel
)

// State is the floor in force for one vault under one recipe, and what it was
// derived from.
type State struct {
	// Floor is the floor in force and Source says where it comes from.
	Floor  float64
	Source Source
	// AutomaticFloor is the floor in force when the manual one is ignored. A floor
	// from the settings file is not automatic, so it does not appear here.
	AutomaticFloor float64
	// IsManual is whether the person set a floor, and ManualFloor its value.
	IsManual    bool
	ManualFloor float64
	// Calibration is nil before the first measurement.
	Calibration *floorport.Calibration
	// Model is the name of the model the floor is for, empty where there is none.
	Model string
}

// Hint returns the starting floor of the model in use, and zero where it has none.
type Hint func(ctx context.Context) float64

// Resolve decides the floor of a vault under a recipe. The first source with an
// answer wins:
//
//  1. the settings file, where it names a floor above zero
//  2. the floor the person set
//  3. the floor a calibration measured
//  4. the model's hint
//  5. the default floor
type Resolve struct {
	store        floorport.Store
	fileFloor    float64
	defaultFloor float64
	hint         Hint
	errorHandler func(error)
}

// NewResolve returns the resolution over a store of floors.
//
// `fileFloor` is the floor of the settings file, zero where it names none, and
// `defaultFloor` the floor where no source answers. A nil store holds no floors,
// a nil hint reports none, and a nil errorHandler ignores the errors of the store.
func NewResolve(store floorport.Store, fileFloor, defaultFloor float64, hint Hint, errorHandler func(error)) Resolve {
	if errorHandler == nil {
		errorHandler = func(error) {}
	}
	return Resolve{store: store, fileFloor: fileFloor, defaultFloor: defaultFloor, hint: hint, errorHandler: errorHandler}
}

// Execute returns the floor in force for the vault under the recipe. A store that
// cannot be read is reported to the error handler and leaves the floor to the
// sources after it.
func (r Resolve) Execute(ctx context.Context, v domain.Vault, recipe string) State {
	var setting floorport.Setting
	if r.store != nil && recipe != "" {
		found, ok, err := r.store.ReadFloor(ctx, v, recipe)
		switch {
		case err != nil:
			r.errorHandler(err)
		case ok:
			setting = found
		}
	}

	state := State{
		IsManual:    setting.IsManual,
		ManualFloor: setting.ManualFloor,
		Calibration: setting.Calibration,
	}
	state.Floor, state.Source = r.getAutomatic(ctx, setting)
	state.AutomaticFloor = state.Floor

	switch {
	case r.fileFloor > 0:
		state.Floor, state.Source = r.fileFloor, SourceFile
	case setting.IsManual && setting.ManualFloor > 0:
		state.Floor, state.Source = setting.ManualFloor, SourceManual
	}
	return state
}

// GetFloor returns the floor in force for the vault under the recipe.
func (r Resolve) GetFloor(ctx context.Context, v domain.Vault, recipe string) float64 {
	return r.Execute(ctx, v, recipe).Floor
}

// getAutomatic is the floor from the calibration, the model's hint or the default.
func (r Resolve) getAutomatic(ctx context.Context, setting floorport.Setting) (float64, Source) {
	if setting.Calibration != nil && setting.Calibration.Floor > 0 {
		return setting.Calibration.Floor, SourceCalibrated
	}
	if r.hint != nil {
		if hint := r.hint(ctx); hint > 0 && hint <= 1 {
			return hint, SourceModel
		}
	}
	return r.defaultFloor, SourceDefault
}

// Get is the floor in force for a vault under the model of an embedder.
type Get struct {
	resolve  Resolve
	embedder embeddingport.Embedder
}

// NewGet returns the floor of the embedder's model. A nil embedder has no model,
// and the floor is then the one that needs none.
func NewGet(resolve Resolve, embedder embeddingport.Embedder) Get {
	return Get{resolve: resolve, embedder: embedder}
}

// Execute returns the floor in force for the vault.
func (u Get) Execute(ctx context.Context, v domain.Vault) State {
	var model embeddingport.EmbeddingModel
	var recipe string
	if u.embedder != nil {
		model = u.embedder.GetModel()
		recipe = model.Recipe()
	}
	state := u.resolve.Execute(ctx, v, recipe)
	state.Model = model.Name
	return state
}
