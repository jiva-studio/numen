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

// PreviewConnect counts the notes of a vault that have another text than the note at the same path of a synced vault.
func (b *Backend) PreviewConnect(ctx context.Context, request api.PreviewConnectRequest) (api.PreviewConnectResponse, error) {
	if !vaultlink.IsRemoteID(request.RemoteID) {
		return api.PreviewConnectResponse{}, catalog.ErrUnknownRemoteVault
	}
	use := preview.NewPreviewConnect(preview.Deps{
		Heads: heads.New(heads.Deps{
			Catalog: b.deps.Catalog, Chain: b.deps.Chain, Revisions: b.deps.Revisions, Crypto: b.deps.Crypto, Identity: b.deps.Identity,
		}),
		Host: b.deps.Host,
	})
	found, err := use.Execute(ctx, preview.Input{VaultID: domain.VaultID(request.VaultID), RemoteID: request.RemoteID})
	return api.PreviewConnectResponse{DifferCount: len(found.Differ)}, err
}
