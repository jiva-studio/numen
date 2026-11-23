package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/heads"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/preview"
)

// GetConnectPreview counts the notes of a vault that have another text than the note at the same path of a synced vault.
func (b *Backend) GetConnectPreview(ctx context.Context, request api.GetConnectPreviewRequest) (api.GetConnectPreviewResponse, error) {
	if !vaultlink.IsRemoteID(request.RemoteID) {
		return api.GetConnectPreviewResponse{}, catalog.ErrUnknownRemoteVault
	}
	use := preview.NewGetConnectPreview(preview.Deps{
		Heads: heads.New(heads.Deps{
			Catalog: b.deps.Catalog, Chain: b.deps.Chain, Revisions: b.deps.Revisions, Crypto: b.deps.Crypto, Identity: b.deps.Identity,
		}),
		Host: b.deps.Host,
	})
	found, err := use.Execute(ctx, preview.Input{VaultID: domain.VaultID(request.VaultID), RemoteID: request.RemoteID})
	return api.GetConnectPreviewResponse{DifferCount: len(found.Differ)}, err
}
