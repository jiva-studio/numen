package indexing

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

// calibrationLog records the vaults a worker requested calibration for.
type calibrationLog struct {
	vaults []domain.VaultID
	err    error
}

func (l *calibrationLog) calibrate(_ context.Context, v domain.Vault) error {
	l.vaults = append(l.vaults, v.ID)
	return l.err
}

// calibratingPass is a pass over a vault of one note that reports into events
// and requests calibration through log.
func calibratingPass(t *testing.T, model embeddingport.Embedder, log *calibrationLog) (*pass, *recorder) {
	t.Helper()
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)
	events := newRecorder()

	p := newPass(made, db, events, model, model.GetModel().Recipe(), v)
	p.worker.calibrate = log.calibrate
	return p, events
}

func TestAPassThatEmbeddedEverythingCalibratesTheFloorOfItsVault(t *testing.T) {
	log := &calibrationLog{}
	p, events := calibratingPass(t, &fakeEmbedder{dims: 64}, log)

	p.embed(t.Context())

	if len(log.vaults) != 1 || log.vaults[0] != p.v.ID {
		t.Fatalf("calibrated %v, want the vault of the pass once", log.vaults)
	}
	if pos := findTask(t, events, CalibratingFloor); pos != nil {
		t.Errorf("a calibration that succeeded is still in the list: %+v", *pos)
	}
}

func TestAPassWithNothingPendingCalibratesAgain(t *testing.T) {
	log := &calibrationLog{}
	p, _ := calibratingPass(t, &fakeEmbedder{dims: 64}, log)

	p.embed(t.Context())
	p.embed(t.Context())

	// The callback decides whether a vault already has a floor, so every pass that
	// leaves nothing pending requests it.
	if len(log.vaults) != 2 {
		t.Errorf("the worker requested calibration %d times, want once for each pass", len(log.vaults))
	}
}

func TestAPassThatCouldNotEmbedCalibratesNothing(t *testing.T) {
	log := &calibrationLog{}
	p, _ := calibratingPass(t, &sulking{fakeEmbedder: &fakeEmbedder{dims: 64}}, log)

	p.embed(t.Context())

	if len(log.vaults) != 0 {
		t.Errorf("calibrated %v after a pass that could not embed", log.vaults)
	}
}

func TestAPassThatWasStoppedCalibratesNothing(t *testing.T) {
	log := &calibrationLog{}
	p, _ := calibratingPass(t, &fakeEmbedder{dims: 64}, log)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	p.embed(ctx)

	if len(log.vaults) != 0 {
		t.Errorf("calibrated %v in a pass that was stopped", log.vaults)
	}
}

func TestACalibrationThatFailedIsReportedAndDoesNotFailThePass(t *testing.T) {
	log := &calibrationLog{err: errors.New("the plugin is not running")}
	p, events := calibratingPass(t, &fakeEmbedder{dims: 64}, log)

	p.embed(t.Context())

	pos := findTask(t, events, CalibratingFloor)
	if pos == nil || pos.Error == "" {
		t.Fatalf("the failure is not in the list: %+v", pos)
	}
	if vectors := findTask(t, events, MakingVectors); vectors != nil {
		t.Errorf("the failed calibration left the vectors in the list: %+v", *vectors)
	}
}

func TestACalibrationThatLaterSucceedsClearsTheTaskList(t *testing.T) {
	log := &calibrationLog{err: errors.New("the plugin is not running")}
	p, events := calibratingPass(t, &fakeEmbedder{dims: 64}, log)
	p.embed(t.Context())
	if findTask(t, events, CalibratingFloor) == nil {
		t.Fatal("the failure was not reported")
	}

	log.err = nil
	p.embed(t.Context())

	if pos := findTask(t, events, CalibratingFloor); pos != nil {
		t.Errorf("the failure stays in the list after a success: %+v", *pos)
	}
}

func TestACalibrationThatWasCancelledIsNotReported(t *testing.T) {
	log := &calibrationLog{err: context.Canceled}
	p, events := calibratingPass(t, &fakeEmbedder{dims: 64}, log)

	p.embed(t.Context())

	if pos := findTask(t, events, CalibratingFloor); pos != nil {
		t.Errorf("a cancelled calibration is in the list: %+v", *pos)
	}
}

func TestAWorkerWithoutACalibrationCallbackEmbedsAsBefore(t *testing.T) {
	made, db := reading(t)
	model := &fakeEmbedder{dims: 64}
	fit(t, db, model)
	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)
	events := newRecorder()

	newPass(made, db, events, model, model.GetModel().Recipe(), v).embed(t.Context())

	if pos := findTask(t, events, MakingVectors); pos != nil {
		t.Errorf("a pass without a calibration is still being done: %+v", *pos)
	}
}
