package search

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/core/usecase/floor"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

type stubVault struct{ v domain.Vault }

func (f stubVault) GetCurrentVault() domain.Vault { return f.v }

var floorVault = domain.Vault{ID: "01VAULT"}

var floorModel = embeddingport.EmbeddingModel{Name: "test-model", Dimensions: 4, From: "test"}

type floorStore struct {
	mu       sync.Mutex
	settings map[string]floorport.Setting
}

func (s *floorStore) ReadFloor(_ context.Context, v domain.Vault, recipe string) (floorport.Setting, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	setting, found := s.settings[string(v.ID)+recipe]
	return setting, found, nil
}

func (s *floorStore) WriteCalibration(_ context.Context, v domain.Vault, recipe string, c floorport.Calibration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	setting := s.settings[string(v.ID)+recipe]
	setting.Calibration = &c
	s.settings[string(v.ID)+recipe] = setting
	return nil
}

func (s *floorStore) WriteManualFloor(_ context.Context, v domain.Vault, recipe string, isManual bool, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	setting := s.settings[string(v.ID)+recipe]
	setting.IsManual, setting.ManualFloor = isManual, value
	s.settings[string(v.ID)+recipe] = setting
	return nil
}

type stubSamples struct {
	titles   int
	passages int
}

func (s stubSamples) GetTitles(context.Context, domain.Vault, int) ([]floorport.Title, error) {
	out := make([]floorport.Title, s.titles)
	for i := range out {
		out[i] = floorport.Title{Source: int64(i + 1), Text: fmt.Sprintf("title %d", i)}
	}
	return out, nil
}

func (s stubSamples) GetPassages(context.Context, domain.Vault, string, int) ([]floorport.Passage, error) {
	out := make([]floorport.Passage, s.passages)
	for i := range out {
		out[i] = floorport.Passage{Source: int64(100 + i%3), Vector: []byte{64, 64, 64, 64}}
	}
	return out, nil
}

type stubNearest struct{ best float64 }

func (n stubNearest) GetBestSimilarity(context.Context, domain.VaultID, string, []float32) (float64, error) {
	return n.best, nil
}

type floorEmbedder struct {
	err    error
	change *embeddingport.EmbeddingModel
	model  embeddingport.EmbeddingModel
}

func (e *floorEmbedder) GetModel() embeddingport.EmbeddingModel { return e.model }
func (e *floorEmbedder) Concurrency() int                       { return 1 }
func (e *floorEmbedder) Close() error                           { return nil }

func (e *floorEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if e.err != nil {
		return nil, e.err
	}
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = []float32{1, 0, 0, 0}
	}
	if e.change != nil {
		e.model = *e.change
	}
	return out, nil
}

type floorFixture struct {
	handler  *Handler
	store    *floorStore
	embedder *floorEmbedder
}

func newFloorFixture(t *testing.T, vault domain.Vault, samples stubSamples, best float64, fileFloor float64) floorFixture {
	t.Helper()
	store := &floorStore{settings: map[string]floorport.Setting{}}
	embedder := &floorEmbedder{model: floorModel}
	scenarios := floor.NewScenarios(floor.Config{
		Store: store, Samples: samples, Nearest: stubNearest{best: best}, Embedder: embedder,
		Clock:     func() time.Time { return time.Unix(1_700_000_000, 0) },
		FileFloor: fileFloor, DefaultFloor: 0.30,
	})
	handler := NewHandler(stubVault{v: vault})
	handler.Floors = &scenarios
	return floorFixture{handler: handler, store: store, embedder: embedder}
}

var enough = stubSamples{titles: 25, passages: 60}

func codeOf(err error) connect.Code {
	var failure *connect.Error
	if errors.As(err, &failure) {
		return failure.Code()
	}
	return connect.CodeUnknown
}

func TestGetFloorStartsAtTheDefaultAndNamesTheModel(t *testing.T) {
	f := newFloorFixture(t, floorVault, enough, 0, 0)

	got, err := f.handler.GetFloor(t.Context(), connect.NewRequest(&v1.GetFloorRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	want := &v1.GetFloorResponse{Floor: 0.30, Source: v1.FloorSource_FLOOR_SOURCE_DEFAULT, Model: "test-model", AutomaticFloor: 0.30}
	if got.Msg.GetFloor() != want.Floor || got.Msg.GetSource() != want.Source || got.Msg.GetModel() != want.Model ||
		got.Msg.GetAutomaticFloor() != want.AutomaticFloor || got.Msg.GetIsManual() || got.Msg.Calibration != nil {
		t.Errorf("got %v, want %v", got.Msg, want)
	}
}

func TestGetFloorReportsTheFileFloorAndTheAutomaticFloor(t *testing.T) {
	f := newFloorFixture(t, floorVault, enough, 0, 0.55)
	if _, err := f.handler.CalibrateFloor(t.Context(), connect.NewRequest(&v1.CalibrateFloorRequest{})); err != nil {
		t.Fatal(err)
	}

	got, err := f.handler.GetFloor(t.Context(), connect.NewRequest(&v1.GetFloorRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetFloor() != 0.55 || got.Msg.GetSource() != v1.FloorSource_FLOOR_SOURCE_FILE {
		t.Errorf("floor %v from %v, want 0.55 from the file", got.Msg.GetFloor(), got.Msg.GetSource())
	}
	if automatic := got.Msg.GetAutomaticFloor(); automatic < 0.50 || automatic > 0.54 {
		t.Errorf("automatic floor %v, want the calibrated 0.52", automatic)
	}
}

func TestCalibrateFloorMeasuresAndReturnsTheNewState(t *testing.T) {
	f := newFloorFixture(t, floorVault, enough, 0, 0)

	got, err := f.handler.CalibrateFloor(t.Context(), connect.NewRequest(&v1.CalibrateFloorRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetOutcome() != v1.CalibrationOutcome_CALIBRATION_OUTCOME_DONE {
		t.Fatalf("outcome %v", got.Msg.GetOutcome())
	}
	state := got.Msg.GetState()
	if state.GetSource() != v1.FloorSource_FLOOR_SOURCE_CALIBRATED || state.GetFloor() < 0.50 || state.GetFloor() > 0.54 {
		t.Errorf("state %v, want the calibrated floor in force", state)
	}
	c := state.GetCalibration()
	if c == nil || c.GetQueryCount() != 25 || c.GetPassageCount() != 60 || len(c.GetHistogram()) != 20 ||
		c.GetCalibratedAt() != 1_700_000_000 || c.GetMedian() < 0.49 || c.GetMedian() > 0.51 {
		t.Errorf("calibration %v", c)
	}
}

func TestCalibrateFloorOfAVaultTooSmallKeepsNothing(t *testing.T) {
	f := newFloorFixture(t, floorVault, stubSamples{titles: 3, passages: 10}, 0, 0)

	got, err := f.handler.CalibrateFloor(t.Context(), connect.NewRequest(&v1.CalibrateFloorRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetOutcome() != v1.CalibrationOutcome_CALIBRATION_OUTCOME_TOO_FEW {
		t.Errorf("outcome %v, want TOO_FEW", got.Msg.GetOutcome())
	}
	if got.Msg.GetState().Calibration != nil || len(f.store.settings) != 0 {
		t.Errorf("state %v, store %v", got.Msg.GetState(), f.store.settings)
	}
}

func TestSetFloorValidatesAManualFloor(t *testing.T) {
	for _, tc := range []struct {
		name     string
		isManual bool
		floor    float64
		want     connect.Code
	}{
		{"zero", true, 0, connect.CodeInvalidArgument},
		{"negative", true, -0.2, connect.CodeInvalidArgument},
		{"above one", true, 1.5, connect.CodeInvalidArgument},
		{"one", true, 1, 0},
		{"inside", true, 0.45, 0},
		{"cleared with a stray value", false, 7, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFloorFixture(t, floorVault, enough, 0, 0)
			_, err := f.handler.SetFloor(t.Context(), connect.NewRequest(&v1.SetFloorRequest{IsManual: tc.isManual, Floor: tc.floor}))
			if tc.want == 0 && err != nil {
				t.Fatalf("got %v", err)
			}
			if tc.want != 0 && codeOf(err) != tc.want {
				t.Fatalf("got code %v from %v, want %v", codeOf(err), err, tc.want)
			}
			if tc.want != 0 && len(f.store.settings) != 0 {
				t.Errorf("an invalid floor was stored: %v", f.store.settings)
			}
		})
	}
}

func TestSetFloorAppliesTheManualFloorAndClearsIt(t *testing.T) {
	f := newFloorFixture(t, floorVault, enough, 0, 0)
	if _, err := f.handler.CalibrateFloor(t.Context(), connect.NewRequest(&v1.CalibrateFloorRequest{})); err != nil {
		t.Fatal(err)
	}

	set, err := f.handler.SetFloor(t.Context(), connect.NewRequest(&v1.SetFloorRequest{IsManual: true, Floor: 0.7}))
	if err != nil {
		t.Fatal(err)
	}
	state := set.Msg.GetState()
	if !state.GetIsManual() || state.GetManualFloor() != 0.7 || state.GetFloor() != 0.7 ||
		state.GetSource() != v1.FloorSource_FLOOR_SOURCE_MANUAL || state.GetAutomaticFloor() > 0.54 {
		t.Errorf("state %v", state)
	}

	cleared, err := f.handler.SetFloor(t.Context(), connect.NewRequest(&v1.SetFloorRequest{IsManual: false}))
	if err != nil {
		t.Fatal(err)
	}
	state = cleared.Msg.GetState()
	if state.GetIsManual() || state.GetManualFloor() != 0 || state.GetSource() != v1.FloorSource_FLOOR_SOURCE_CALIBRATED {
		t.Errorf("state %v, want the calibrated floor back in force", state)
	}
}

func TestProbeFloorComparesTheBestPassageWithTheFloor(t *testing.T) {
	for _, tc := range []struct {
		best     float64
		wantPass bool
	}{{0.25, false}, {0.30, true}, {0.9, true}} {
		f := newFloorFixture(t, floorVault, enough, tc.best, 0)
		got, err := f.handler.ProbeFloor(t.Context(), connect.NewRequest(&v1.ProbeFloorRequest{Query: "entropy"}))
		if err != nil {
			t.Fatal(err)
		}
		if got.Msg.GetBestSimilarity() != tc.best || got.Msg.GetFloor() != 0.30 || got.Msg.GetDoesPass() != tc.wantPass {
			t.Errorf("best %v: got %v", tc.best, got.Msg)
		}
	}
}

func TestFloorRequestsMapEachFailureToItsCode(t *testing.T) {
	changed := embeddingport.EmbeddingModel{Name: "other", Dimensions: 4, From: "test"}
	for _, tc := range []struct {
		name string
		run  func(f floorFixture) error
		want connect.Code
	}{
		{"calibrate with the plugin down", func(f floorFixture) error {
			f.embedder.err = errors.New("the plugin is not running")
			_, err := f.handler.CalibrateFloor(t.Context(), connect.NewRequest(&v1.CalibrateFloorRequest{}))
			return err
		}, connect.CodeUnavailable},
		{"calibrate while the model changes", func(f floorFixture) error {
			f.embedder.change = &changed
			_, err := f.handler.CalibrateFloor(t.Context(), connect.NewRequest(&v1.CalibrateFloorRequest{}))
			return err
		}, connect.CodeFailedPrecondition},
		{"probe with the plugin down", func(f floorFixture) error {
			f.embedder.err = errors.New("no key")
			_, err := f.handler.ProbeFloor(t.Context(), connect.NewRequest(&v1.ProbeFloorRequest{Query: "q"}))
			return err
		}, connect.CodeUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFloorFixture(t, floorVault, enough, 0, 0)
			if err := tc.run(f); codeOf(err) != tc.want {
				t.Errorf("got code %v from %v, want %v", codeOf(err), err, tc.want)
			}
			if len(f.store.settings) != 0 {
				t.Errorf("a failed request stored %v", f.store.settings)
			}
		})
	}
}

func TestFloorRequestsOfAWindowWithoutAVaultFail(t *testing.T) {
	f := newFloorFixture(t, domain.Vault{}, enough, 0, 0)
	ctx := t.Context()

	if _, err := f.handler.CalibrateFloor(ctx, connect.NewRequest(&v1.CalibrateFloorRequest{})); codeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("calibrate: %v", err)
	}
	if _, err := f.handler.SetFloor(ctx, connect.NewRequest(&v1.SetFloorRequest{IsManual: true, Floor: 0.5})); codeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("set: %v", err)
	}
	if _, err := f.handler.ProbeFloor(ctx, connect.NewRequest(&v1.ProbeFloorRequest{Query: "q"})); codeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("probe: %v", err)
	}
	if _, err := f.handler.GetFloor(ctx, connect.NewRequest(&v1.GetFloorRequest{})); err != nil {
		t.Errorf("get: %v", err)
	}
}

func TestFloorRequestsOfAWindowWithoutFloorsFail(t *testing.T) {
	handler := NewHandler(stubVault{v: floorVault})
	ctx := t.Context()

	if _, err := handler.GetFloor(ctx, connect.NewRequest(&v1.GetFloorRequest{})); codeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("get: %v", err)
	}
	if _, err := handler.CalibrateFloor(ctx, connect.NewRequest(&v1.CalibrateFloorRequest{})); codeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("calibrate: %v", err)
	}
}

func TestFloorRequestsWithoutAModelAreUnavailable(t *testing.T) {
	store := &floorStore{settings: map[string]floorport.Setting{}}
	scenarios := floor.NewScenarios(floor.Config{
		Store: store, Samples: enough, Nearest: stubNearest{}, Embedder: nil,
		Clock: time.Now, DefaultFloor: 0.30,
	})
	handler := NewHandler(stubVault{v: floorVault})
	handler.Floors = &scenarios
	ctx := t.Context()

	if _, err := handler.CalibrateFloor(ctx, connect.NewRequest(&v1.CalibrateFloorRequest{})); codeOf(err) != connect.CodeUnavailable {
		t.Errorf("calibrate: %v", err)
	}
	if _, err := handler.ProbeFloor(ctx, connect.NewRequest(&v1.ProbeFloorRequest{Query: "q"})); codeOf(err) != connect.CodeUnavailable {
		t.Errorf("probe: %v", err)
	}
	if _, err := handler.SetFloor(ctx, connect.NewRequest(&v1.SetFloorRequest{IsManual: true, Floor: 0.5})); codeOf(err) != connect.CodeUnavailable {
		t.Errorf("set: %v", err)
	}
	got, err := handler.GetFloor(ctx, connect.NewRequest(&v1.GetFloorRequest{}))
	if err != nil || got.Msg.GetModel() != "" || got.Msg.GetFloor() != 0.30 {
		t.Errorf("get: %v, %v", got, err)
	}
}

func TestEveryFloorSourceHasItsOwnWireValue(t *testing.T) {
	want := map[floor.Source]v1.FloorSource{
		floor.SourceFile:       v1.FloorSource_FLOOR_SOURCE_FILE,
		floor.SourceManual:     v1.FloorSource_FLOOR_SOURCE_MANUAL,
		floor.SourceCalibrated: v1.FloorSource_FLOOR_SOURCE_CALIBRATED,
		floor.SourceModel:      v1.FloorSource_FLOOR_SOURCE_MODEL,
		floor.SourceDefault:    v1.FloorSource_FLOOR_SOURCE_DEFAULT,
	}
	wireSet := map[v1.FloorSource]bool{}
	for source, wire := range want {
		if got := convertFloorSourceToWire(source); got != wire {
			t.Errorf("source %v = %v, want %v", source, got, wire)
		}
		wireSet[wire] = true
	}
	for value := range v1.FloorSource_name {
		if wire := v1.FloorSource(value); wire != v1.FloorSource_FLOOR_SOURCE_UNSPECIFIED && !wireSet[wire] {
			t.Errorf("%v is on the wire and no source is written as it", wire)
		}
	}
}

func TestEveryCalibrationOutcomeHasItsOwnWireValue(t *testing.T) {
	want := map[floor.Outcome]v1.CalibrationOutcome{
		floor.Calibrated: v1.CalibrationOutcome_CALIBRATION_OUTCOME_DONE,
		floor.TooFew:     v1.CalibrationOutcome_CALIBRATION_OUTCOME_TOO_FEW,
	}
	wireSet := map[v1.CalibrationOutcome]bool{}
	for outcome, wire := range want {
		if got := convertCalibrationOutcomeToWire(outcome); got != wire {
			t.Errorf("outcome %v = %v, want %v", outcome, got, wire)
		}
		wireSet[wire] = true
	}
	for value := range v1.CalibrationOutcome_name {
		if wire := v1.CalibrationOutcome(value); wire != v1.CalibrationOutcome_CALIBRATION_OUTCOME_UNSPECIFIED && !wireSet[wire] {
			t.Errorf("%v is on the wire and no outcome is written as it", wire)
		}
	}
}
