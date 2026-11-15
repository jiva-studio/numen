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

func (a *App) newRegistry(built *parts) (*tools.Registry, error) {
	registry := tools.New()
	previewSpec := tools.Spec{
		Name: "saved_search_preview", Title: "Preview a saved search", Description: previewDescription, Effect: tools.ReadOnly,
		Instructions: []string{quotesSentence, exampleSentence, resultLinkSentence, truncatedSentence},
		Requires:     []string{host.PermissionIndexRead},
	}
	err := tools.Add(registry, previewSpec, func(ctx context.Context, in api.PreviewInput) (api.PreviewOutput, error) {
		id := tools.GetVault(ctx)
		if id == "" {
			return api.PreviewOutput{}, errNoVault
		}
		vault := built.hosts.Open(domain.VaultID(id))
		deps := preview.Dependencies{Search: vault.Search, Notes: vault.Lookup, Reranker: built.reranker}
		return preview.New(deps, a.settings.GetConfig()).Execute(ctx, in)
	})
	if err != nil {
		return nil, err
	}
	createSpec := tools.Spec{
		Name: "saved_search_create", Title: "Create a saved search", Description: createDescription, Effect: tools.ReadWrite,
		Instructions: []string{quotesSentence, noteLinkSentence},
		Requires:     []string{host.PermissionIndexRead},
	}
	err = tools.Add(registry, createSpec, func(ctx context.Context, in api.CreateInput) (api.SavedOutput, error) {
		id := tools.GetVault(ctx)
		if id == "" {
			return api.SavedOutput{}, errNoVault
		}
		return built.notes.Create(ctx, domain.VaultID(id), in)
	})
	if err != nil {
		return nil, err
	}
	updateSpec := tools.Spec{
		Name: "saved_search_update", Title: "Update a saved search", Description: updateDescription, Effect: tools.ReadWrite,
		Instructions: []string{quotesSentence, noteLinkSentence},
		Requires:     []string{host.PermissionIndexRead},
	}
	err = tools.Add(registry, updateSpec, func(ctx context.Context, in api.UpdateInput) (api.SavedOutput, error) {
		id := tools.GetVault(ctx)
		if id == "" {
			return api.SavedOutput{}, errNoVault
		}
		return built.notes.Update(ctx, domain.VaultID(id), in)
	})
	if err != nil {
		return nil, err
	}
	return registry, nil
}
