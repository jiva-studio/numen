// Package floorwire converts between the search relevance domain values and the JSON messages of api.
package floorwire

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/calibration"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/relevance"
)

// ErrNotMounted is a call that reached the service before it was mounted.
var ErrNotMounted = server.NewAPIError(server.CodeUnavailable, "the plugin is not ready")

// ConvertError translates a domain or host error into the corresponding APIError.
func ConvertError(err error) error {
	switch {
	case errors.Is(err, calibration.ErrNotStarted):
		return ErrNotMounted
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

// ConvertState converts a floor state into its wire message.
func ConvertState(state relevance.State) api.GetFloorResponse {
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

// ConvertCalibrate converts a calibration outcome and state into its wire message.
func ConvertCalibrate(outcome relevance.Outcome, state relevance.State) api.CalibrateFloorResponse {
	return api.CalibrateFloorResponse{
		Outcome: convertOutcome(outcome),
		State:   ConvertState(state),
	}
}

// ConvertProbe converts a probe result into its wire message.
func ConvertProbe(result relevance.ProbeResult) api.ProbeFloorResponse {
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
