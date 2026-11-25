package filebridge

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// CapabilityResolver finds the enabled plugin that provides a capability.
type CapabilityResolver interface {
	ResolveCapability(capability string) (manifest.DiscoveredPlugin, error)
}

// Provider calls the FileProviderService of the enabled plugin with the capability files.provider, which
// it activates for the call.
type Provider struct {
	activator bridge.Activator
	plugins   CapabilityResolver
}

// NewProvider returns the provider over the plugins the activator starts.
func NewProvider(activator bridge.Activator, plugins CapabilityResolver) *Provider {
	return &Provider{activator: activator, plugins: plugins}
}

// FetchFile asks the plugin to fill the absent file at path of the vault.
func (p *Provider) FetchFile(ctx context.Context, v domain.Vault, path string) error {
	return call(ctx, p, "fetch file", func(client numenv1connect.FileProviderServiceClient) error {
		_, err := client.FetchFile(ctx, connect.NewRequest(&v1.FileProviderServiceFetchFileRequest{VaultId: string(v.ID), Path: path}))
		return err
	})
}

// DropDownload asks the plugin to remove the download of the file at path of the vault.
func (p *Provider) DropDownload(ctx context.Context, v domain.Vault, path string) error {
	return call(ctx, p, "drop download", func(client numenv1connect.FileProviderServiceClient) error {
		_, err := client.DropDownload(ctx, connect.NewRequest(&v1.FileProviderServiceDropDownloadRequest{VaultId: string(v.ID), Path: path}))
		return err
	})
}

// GetFetchProgress asks the plugin how far the download of the file at path has come.
func (p *Provider) GetFetchProgress(ctx context.Context, v domain.Vault, path string) (file.DownloadProgress, error) {
	var progress file.DownloadProgress
	err := call(ctx, p, "get fetch progress", func(client numenv1connect.FileProviderServiceClient) error {
		answer, err := client.GetFetchProgress(ctx, connect.NewRequest(&v1.GetFetchProgressRequest{Path: path}))
		if err != nil {
			return err
		}
		progress = file.DownloadProgress{IsKnown: answer.Msg.GetIsKnown(), BytesDone: answer.Msg.GetBytesDone(), BytesTotal: answer.Msg.GetBytesTotal()}
		return nil
	})
	return progress, err
}

func call(ctx context.Context, p *Provider, why string, run func(numenv1connect.FileProviderServiceClient) error) error {
	found, err := p.plugins.ResolveCapability(bridge.CapabilityFilesProvider)
	if err != nil {
		return err
	}
	client, lease, err := bridge.Open(ctx, p.activator, found.Manifest.ID, why, numenv1connect.NewFileProviderServiceClient)
	if err != nil {
		return err
	}
	defer lease.Release()
	return run(client)
}
