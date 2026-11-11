package service

import (
	"context"

	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/floorwire"
)

func (a *App) getFloor(ctx context.Context, _ api.GetFloorRequest) (api.GetFloorResponse, error) {
	state, err := a.calibration.Get(ctx)
	if err != nil {
		return api.GetFloorResponse{}, floorwire.ConvertError(err)
	}
	return floorwire.ConvertState(state), nil
}

func (a *App) calibrateFloor(ctx context.Context, _ api.CalibrateFloorRequest) (api.CalibrateFloorResponse, error) {
	outcome, state, err := a.calibration.Calibrate(ctx)
	if err != nil {
		return api.CalibrateFloorResponse{}, floorwire.ConvertError(err)
	}
	return floorwire.ConvertCalibrate(outcome, state), nil
}

func (a *App) setFloor(ctx context.Context, req api.SetFloorRequest) (api.SetFloorResponse, error) {
	state, err := a.calibration.Set(ctx, req.IsManual, req.Floor)
	if err != nil {
		return api.SetFloorResponse{}, floorwire.ConvertError(err)
	}
	return api.SetFloorResponse{State: floorwire.ConvertState(state)}, nil
}

func (a *App) probeFloor(ctx context.Context, req api.ProbeFloorRequest) (api.ProbeFloorResponse, error) {
	result, err := a.calibration.Probe(ctx, req.Query)
	if err != nil {
		return api.ProbeFloorResponse{}, floorwire.ConvertError(err)
	}
	return floorwire.ConvertProbe(result), nil
}
