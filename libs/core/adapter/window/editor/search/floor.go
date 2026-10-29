package search

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/usecase/floor"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// errNoFloors is a window that was put together without the floor use cases.
var errNoFloors = errors.New("this window has no search floor")

// errNoVault is a floor asked for on a window with no vault open.
var errNoVault = errors.New("the window has no vault")

// GetFloor returns the floor in force for the vault in the window.
func (h *Handler) GetFloor(
	ctx context.Context, _ *connect.Request[v1.GetFloorRequest],
) (*connect.Response[v1.GetFloorResponse], error) {
	if h.Floors == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoFloors)
	}
	return connect.NewResponse(convertFloorStateToWire(h.Floors.Get.Execute(ctx, h.Vaults.GetCurrentVault()))), nil
}

// CalibrateFloor measures the floor on the vault in the window and keeps it.
func (h *Handler) CalibrateFloor(
	ctx context.Context, _ *connect.Request[v1.CalibrateFloorRequest],
) (*connect.Response[v1.CalibrateFloorResponse], error) {
	showing, err := h.getFloorVault()
	if err != nil {
		return nil, err
	}
	outcome, _, err := h.Floors.Calibrate.Execute(ctx, showing)
	if err != nil {
		return nil, convertFloorErrorToWire(err)
	}
	return connect.NewResponse(&v1.CalibrateFloorResponse{
		Outcome: convertCalibrationOutcomeToWire(outcome),
		State:   convertFloorStateToWire(h.Floors.Get.Execute(ctx, showing)),
	}), nil
}

// SetFloor puts the floor the person chose in force, or returns to the
// automatic one.
func (h *Handler) SetFloor(
	ctx context.Context, r *connect.Request[v1.SetFloorRequest],
) (*connect.Response[v1.SetFloorResponse], error) {
	showing, err := h.getFloorVault()
	if err != nil {
		return nil, err
	}
	if err := h.Floors.Set.Execute(ctx, showing, r.Msg.GetIsManual(), r.Msg.GetFloor()); err != nil {
		return nil, convertFloorErrorToWire(err)
	}
	return connect.NewResponse(&v1.SetFloorResponse{
		State: convertFloorStateToWire(h.Floors.Get.Execute(ctx, showing)),
	}), nil
}

// ProbeFloor measures how near the best passage of the vault stands to a query.
func (h *Handler) ProbeFloor(
	ctx context.Context, r *connect.Request[v1.ProbeFloorRequest],
) (*connect.Response[v1.ProbeFloorResponse], error) {
	showing, err := h.getFloorVault()
	if err != nil {
		return nil, err
	}
	result, err := h.Floors.Probe.Execute(ctx, showing, r.Msg.GetQuery())
	if err != nil {
		return nil, convertFloorErrorToWire(err)
	}
	return connect.NewResponse(&v1.ProbeFloorResponse{
		BestSimilarity: result.BestSimilarity,
		Floor:          result.Floor,
		DoesPass:       result.DoesPass,
	}), nil
}

// getFloorVault is the vault the window shows, or the error of a window that
// cannot answer a floor request.
func (h *Handler) getFloorVault() (domain.Vault, error) {
	if h.Floors == nil {
		return domain.Vault{}, connect.NewError(connect.CodeFailedPrecondition, errNoFloors)
	}
	showing := h.Vaults.GetCurrentVault()
	if showing.ID == "" {
		return domain.Vault{}, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	return showing, nil
}

// convertFloorErrorToWire is the error a client receives for a failed floor request.
func convertFloorErrorToWire(err error) error {
	switch {
	case errors.Is(err, floor.ErrInvalidFloor):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, floor.ErrRecipeChanged):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, floor.ErrNoModel), errors.Is(err, floor.ErrEmbedding):
		return connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func convertFloorStateToWire(state floor.State) *v1.GetFloorResponse {
	out := &v1.GetFloorResponse{
		Floor:          state.Floor,
		Source:         convertFloorSourceToWire(state.Source),
		Model:          state.Model,
		AutomaticFloor: state.AutomaticFloor,
		IsManual:       state.IsManual,
		ManualFloor:    state.ManualFloor,
	}
	if c := state.Calibration; c != nil {
		histogram := make([]int32, len(c.Histogram))
		for i, count := range c.Histogram {
			histogram[i] = int32(count)
		}
		out.Calibration = &v1.FloorCalibration{
			Median:       c.Median,
			Spread:       c.Spread,
			Histogram:    histogram,
			QueryCount:   int32(c.QueryCount),
			PassageCount: int32(c.PassageCount),
			CalibratedAt: c.CalibratedTime.Unix(),
		}
	}
	return out
}

func convertFloorSourceToWire(source floor.Source) v1.FloorSource {
	switch source {
	case floor.SourceFile:
		return v1.FloorSource_FLOOR_SOURCE_FILE
	case floor.SourceManual:
		return v1.FloorSource_FLOOR_SOURCE_MANUAL
	case floor.SourceCalibrated:
		return v1.FloorSource_FLOOR_SOURCE_CALIBRATED
	case floor.SourceModel:
		return v1.FloorSource_FLOOR_SOURCE_MODEL
	default:
		return v1.FloorSource_FLOOR_SOURCE_DEFAULT
	}
}

func convertCalibrationOutcomeToWire(outcome floor.Outcome) v1.CalibrationOutcome {
	switch outcome {
	case floor.Calibrated:
		return v1.CalibrationOutcome_CALIBRATION_OUTCOME_DONE
	case floor.TooFew:
		return v1.CalibrationOutcome_CALIBRATION_OUTCOME_TOO_FEW
	default:
		return v1.CalibrationOutcome_CALIBRATION_OUTCOME_UNSPECIFIED
	}
}
