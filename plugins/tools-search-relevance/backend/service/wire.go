package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/calibration"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/relevance"
)

// errNotMounted is a call that reached the service before it was mounted.
var errNotMounted = server.NewAPIError(server.CodeUnavailable, "the plugin is not ready")

func convertError(err error) error {
	switch {
	case errors.Is(err, calibration.ErrNotStarted):
		return errNotMounted
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
			ChunkCount:   c.ChunkCount,
			CalibratedAt: c.CalibratedTime.Unix(),
		}
	}
	return out
}

func convertCalibrate(outcome relevance.Outcome, state relevance.State) api.CalibrateFloorResponse {
	return api.CalibrateFloorResponse{
		Outcome: convertOutcome(outcome),
		State:   convertState(state),
	}
}

func convertProbe(result relevance.ProbeResult) api.ProbeFloorResponse {
	return api.ProbeFloorResponse{
		BestSimilarity: result.BestSimilarity,
		Floor:          result.Floor,
		DoesPass:       result.DoesPass,
	}
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
