package floor

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// AutoCalibrate calibrates a vault once, when its embedding is complete.
type AutoCalibrate struct {
	store     floorport.Store
	calibrate Calibrate
	embedder  embeddingport.Embedder
}

// NewAutoCalibrate returns the automatic calibration over the embedder given.
func NewAutoCalibrate(store floorport.Store, calibrate Calibrate, embedder embeddingport.Embedder) AutoCalibrate {
	return AutoCalibrate{store: store, calibrate: calibrate, embedder: embedder}
}

// Execute calibrates the vault under the model in use when nothing is kept for
// the pair: no calibration and no manual floor. A vault with too few notes to
// measure is unchanged and is calibrated again at the next pass.
func (u AutoCalibrate) Execute(ctx context.Context, v domain.Vault) error {
	if u.embedder == nil {
		return nil
	}
	setting, found, err := u.store.ReadFloor(ctx, v, u.embedder.GetModel().Recipe())
	if err != nil {
		return err
	}
	if found && (setting.IsManual || setting.Calibration != nil) {
		return nil
	}
	_, _, err = u.calibrate.Execute(ctx, v)
	return err
}

// Scenarios are the floor use cases of one embedder.
type Scenarios struct {
	Resolve       Resolve
	Get           Get
	Calibrate     Calibrate
	Set           Set
	Probe         Probe
	AutoCalibrate AutoCalibrate
}

// Config is what the floor use cases are made from.
type Config struct {
	Store    floorport.Store
	Samples  floorport.Samples
	Nearest  floorport.Nearest
	Embedder embeddingport.Embedder
	Clock    machineport.Clock
	// FileFloor is the floor of the settings file, zero where it names none, and
	// DefaultFloor the floor where no source answers.
	FileFloor    float64
	DefaultFloor float64
	// Hint reports the starting floor of the model, and may be nil.
	Hint Hint
	// ErrorHandler receives the errors of a store that cannot be read, and may be nil.
	ErrorHandler func(error)
}

// NewScenarios returns the floor use cases over the config given.
func NewScenarios(cfg Config) Scenarios {
	resolve := NewResolve(cfg.Store, cfg.FileFloor, cfg.DefaultFloor, cfg.Hint, cfg.ErrorHandler)
	calibrate := NewCalibrate(cfg.Samples, cfg.Store, cfg.Embedder, cfg.Clock)
	return Scenarios{
		Resolve:       resolve,
		Get:           NewGet(resolve, cfg.Embedder),
		Calibrate:     calibrate,
		Set:           NewSet(cfg.Store, cfg.Embedder),
		Probe:         NewProbe(cfg.Nearest, resolve, cfg.Embedder),
		AutoCalibrate: NewAutoCalibrate(cfg.Store, calibrate, cfg.Embedder),
	}
}
