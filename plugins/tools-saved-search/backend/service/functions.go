package service

import (
	"context"

	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
)

func listSearchFunctions(ctx context.Context, built *parts) (api.ListSearchFunctionsResponse, error) {
	vault, err := getShownVault(ctx, built)
	if err != nil {
		return api.ListSearchFunctionsResponse{}, err
	}
	catalog, err := built.hosts.Open(vault).Search.ListSearchFunctions(ctx)
	if err != nil {
		return api.ListSearchFunctionsResponse{}, err
	}
	functions := make([]api.SearchFunction, len(catalog.Functions))
	for i, f := range catalog.Functions {
		functions[i] = api.SearchFunction{
			Name: f.Name, Kind: f.Kind, Signatures: append([]string{}, f.Signatures...), Description: f.Description, Example: f.Example,
		}
	}
	return api.ListSearchFunctionsResponse{Functions: functions}, nil
}
