package service

import (
	"context"

	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/api"
)

func (a *App) getFloor(ctx context.Context, _ api.GetFloorRequest) (api.GetFloorResponse, error) {
	state, err := a.calibration.Get(ctx)
	if err != nil {
		return api.GetFloorResponse{}, convertError(err)
	}
	return convertState(state), nil
}

func (a *App) calibrateFloor(ctx context.Context, _ api.CalibrateFloorRequest) (api.CalibrateFloorResponse, error) {
	outcome, state, err := a.calibration.Calibrate(ctx)
	if err != nil {
		return api.CalibrateFloorResponse{}, convertError(err)
	}
	return convertCalibrate(outcome, state), nil
}

func (a *App) setFloor(ctx context.Context, req api.SetFloorRequest) (api.SetFloorResponse, error) {
	state, err := a.calibration.Set(ctx, req.IsManual, req.Floor)
	if err != nil {
		return api.SetFloorResponse{}, convertError(err)
	}
	return api.SetFloorResponse{State: convertState(state)}, nil
}

func (a *App) probeFloor(ctx context.Context, req api.ProbeFloorRequest) (api.ProbeFloorResponse, error) {
	result, err := a.calibration.Probe(ctx, req.Query)
	if err != nil {
		return api.ProbeFloorResponse{}, convertError(err)
	}
	return convertProbe(result), nil
}
