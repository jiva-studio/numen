package filebridge

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
)

// CapabilityResolver finds the enabled plugin that provides a capability.
type CapabilityResolver interface {
	ResolveCapability(capability string) (manifest.Discovered, error)
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

func call(ctx context.Context, p *Provider, why string, run func(numenv1connect.FileProviderServiceClient) error) (err error) {
	found, err := p.plugins.ResolveCapability(bridge.CapabilityFilesProvider)
	if err != nil {
		return err
	}
	client, lease, err := bridge.Open(ctx, p.activator, found.Manifest.ID, why, numenv1connect.NewFileProviderServiceClient)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	return run(client)
}
