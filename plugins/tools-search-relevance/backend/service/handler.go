package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/relevance"
)

func (a *App) begin() (*relevance.Floors, error) {
	floors, err := a.calibration.GetFloors()
	if err != nil {
		return nil, errNotMounted
	}
	return floors, nil
}

func (a *App) getFloor(ctx context.Context, _ api.GetFloorRequest) (api.GetFloorResponse, error) {
	floors, err := a.begin()
	if err != nil {
		return api.GetFloorResponse{}, err
	}
	state, err := floors.Get(ctx, "")
	if err != nil {
		return api.GetFloorResponse{}, convertError(err)
	}
	return convertState(state), nil
}

func (a *App) calibrateFloor(ctx context.Context, _ api.CalibrateFloorRequest) (api.CalibrateFloorResponse, error) {
	floors, err := a.begin()
	if err != nil {
		return api.CalibrateFloorResponse{}, err
	}
	outcome, state, err := floors.Calibrate(ctx, "")
	if err != nil {
		return api.CalibrateFloorResponse{}, convertError(err)
	}
	return api.CalibrateFloorResponse{Outcome: convertOutcome(outcome), State: convertState(state)}, nil
}

func (a *App) setFloor(ctx context.Context, req api.SetFloorRequest) (api.SetFloorResponse, error) {
	floors, err := a.begin()
	if err != nil {
		return api.SetFloorResponse{}, err
	}
	state, err := floors.Set(ctx, "", req.IsManual, req.Floor)
	if err != nil {
		return api.SetFloorResponse{}, convertError(err)
	}
	return api.SetFloorResponse{State: convertState(state)}, nil
}

func (a *App) probeFloor(ctx context.Context, req api.ProbeFloorRequest) (api.ProbeFloorResponse, error) {
	floors, err := a.begin()
	if err != nil {
		return api.ProbeFloorResponse{}, err
	}
	result, err := floors.Probe(ctx, "", req.Query)
	if err != nil {
		return api.ProbeFloorResponse{}, convertError(err)
	}
	return api.ProbeFloorResponse{BestSimilarity: result.BestSimilarity, Floor: result.Floor, DoesPass: result.DoesPass}, nil
}

func convertError(err error) error {
	switch {
	case errors.Is(err, host.ErrInvalidFloor):
		return server.NewAPIError(server.CodeInvalidArgument, err.Error())
	case errors.Is(err, relevance.ErrRecipeChanged), errors.Is(err, host.ErrRecipeMismatch), errors.Is(err, host.ErrNoVault):
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	case errors.Is(err, host.ErrNoModel), errors.Is(err, host.ErrEmbeddingFailed), errors.Is(err, context.Canceled):
		return server.NewAPIError(server.CodeUnavailable, err.Error())
	case errors.Is(err, host.ErrPermissionDenied):
		return server.NewAPIError(server.CodePermissionDenied, err.Error())
	case errors.Is(err, host.ErrUnknownVault):
		return server.NewAPIError(server.CodeNotFound, err.Error())
	default:
		return server.NewAPIError(server.CodeInternal, err.Error())
	}
}

func convertState(state relevance.State) api.GetFloorResponse {
	out := api.GetFloorResponse{
		Floor:          state.Floor,
		Source:         convertSource(state.Source),
		Model:          state.Model,
		AutomaticFloor: state.AutomaticFloor,
		IsManual:       state.IsManual,
		ManualFloor:    state.ManualFloor,
		Calibration:    nil,
	}
	if c := state.Calibration; c != nil {
		out.Calibration = &api.Calibration{
			Median:       c.Median,
			Spread:       c.Spread,
			Histogram:    c.Histogram,
			QueryCount:   c.QueryCount,
			PassageCount: c.PassageCount,
			CalibratedAt: c.CalibratedTime.Unix(),
		}
	}
	return out
}

func convertSource(source relevance.Source) api.Source {
	switch source {
	case relevance.SourceManual:
		return api.SourceManual
	case relevance.SourceCalibrated:
		return api.SourceCalibrated
	case relevance.SourceModel:
		return api.SourceModel
	case relevance.SourceDefault:
		return api.SourceDefault
	}
	return api.SourceDefault
}

func convertOutcome(outcome relevance.Outcome) api.Outcome {
	switch outcome {
	case relevance.Calibrated:
		return api.OutcomeCalibrated
	case relevance.TooFew:
		return api.OutcomeTooFew
	}
	return api.OutcomeTooFew
}
