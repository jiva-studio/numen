package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/usecase/preview"
)

var errNoVault = errors.New("the call names no vault")

// NewRegistry returns the registry of saved_search_preview. open gives what a preview reads of the vault
// a call names, and config the numbers of the settings at the time of the call.
func NewRegistry(open func(domain.VaultID) preview.Dependencies, config func() preview.Config) (*tools.Registry, error) {
	registry := tools.New()
	spec := tools.Spec{
		Name: "saved_search_preview", Title: "Preview a saved search", Description: previewDescription, Effect: tools.ReadOnly,
		Instructions: []string{quotesSentence, exampleSentence, resultLinkSentence, truncatedSentence},
		Requires:     []string{host.PermissionIndexRead},
	}
	err := tools.Add(registry, spec, func(ctx context.Context, in api.PreviewInput) (api.PreviewOutput, error) {
		id := tools.GetVault(ctx)
		if id == "" {
			return api.PreviewOutput{
				Query: in.Query, Results: []api.PreviewResult{}, Order: api.OrderFused, Floor: 0, CandidateCount: 0,
				Warnings: nil, Errors: nil,
			}, errNoVault
		}
		return preview.New(open(domain.VaultID(id)), config()).Execute(ctx, in)
	})
	if err != nil {
		return nil, err
	}
	return registry, nil
}
