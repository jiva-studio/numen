package service

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/refresh"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/savedsearch"
)

func getResults(ctx context.Context, built *parts, req api.GetResultsRequest) (api.GetResultsResponse, error) {
	vault, err := getShownVault(ctx, built)
	if err != nil {
		return api.GetResultsResponse{}, err
	}
	state, err := built.runner.ReadState(ctx, vault, req.Path)
	if err != nil {
		return api.GetResultsResponse{}, convertNoteError(req.Path, err)
	}
	return newResultsResponse(req.Path, state), nil
}

func watchResults(ctx context.Context, built *parts, req api.WatchResultsRequest, send func(api.ResultsChange) error) error {
	vault, err := getShownVault(ctx, built)
	if err != nil {
		return err
	}
	changes, stop := built.runner.Watch(vault, req.Path)
	defer stop()
	state, err := built.runner.ReadState(ctx, vault, req.Path)
	if err != nil {
		return convertNoteError(req.Path, err)
	}
	computedAt := time.Time{}
	if state.HasRecord {
		computedAt = state.Record.ComputedAt
	}
	if err := send(api.ResultsChange{ComputedAt: formatTime(computedAt), IsRunning: state.IsRunning}); err != nil {
		return err
	}
	built.runner.StartRunDue(ctx, vault)
	for {
		select {
		case <-ctx.Done():
			return nil
		case change, isOpen := <-changes:
			if !isOpen {
				return nil
			}
			if err := send(api.ResultsChange{ComputedAt: formatTime(change.ComputedAt), IsRunning: change.IsRunning}); err != nil {
				return err
			}
		}
	}
}

func updateSearch(ctx context.Context, built *parts, req api.UpdateSearchRequest) (api.UpdateSearchResponse, error) {
	vault, err := getShownVault(ctx, built)
	if err != nil {
		return api.UpdateSearchResponse{}, err
	}
	problems, err := built.notes.Write(ctx, vault, req.Path, req.Query)
	if err != nil {
		return api.UpdateSearchResponse{}, convertNoteError(req.Path, err)
	}
	return api.UpdateSearchResponse{Errors: convertProblems(problems)}, nil
}

func getShownVault(ctx context.Context, built *parts) (domain.VaultID, error) {
	vault, err := built.hosts.Window.GetVault(ctx)
	if errors.Is(err, host.ErrNoVault) {
		return "", server.NewAPIError(server.CodeFailedPrecondition, "the window shows no vault")
	}
	if err != nil {
		return "", err
	}
	return vault.ID, nil
}

func convertNoteError(notePath string, err error) error {
	switch {
	case errors.Is(err, host.ErrNotExist):
		return server.NewAPIError(server.CodeNotFound, "there is no note at "+notePath)
	case errors.Is(err, refresh.ErrNotASearch):
		return server.NewAPIError(server.CodeInvalidArgument, notePath+" is not a saved search")
	case errors.Is(err, savedsearch.ErrChanged):
		return server.NewAPIError(server.CodeFailedPrecondition, err.Error())
	case errors.Is(err, host.ErrNoVault):
		return server.NewAPIError(server.CodeFailedPrecondition, "the window shows no vault")
	}
	return err
}

func newResultsResponse(notePath string, state refresh.State) api.GetResultsResponse {
	record := state.Record
	response := api.GetResultsResponse{
		Path: notePath, Title: state.Definition.Title, Query: state.Definition.Query,
		ComputedAt: formatTime(record.ComputedAt), FailedAt: "", Failure: record.Failure, IsStale: state.IsStale,
		IsRunning: state.IsRunning, Order: api.Order(record.Order), Results: []api.Result{}, Warnings: []string{}, Errors: []api.Problem{},
	}
	if record.FailedAt != nil {
		response.FailedAt = formatTime(*record.FailedAt)
	}
	response.Results = append(response.Results, convertResults(record.Results)...)
	response.Warnings = append(response.Warnings, record.Warnings...)
	response.Errors = append(response.Errors, convertProblems(state.Problems)...)
	return response
}

func formatTime(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.UTC().Format(time.RFC3339)
}
