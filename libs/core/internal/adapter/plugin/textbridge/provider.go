package textbridge

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	textport "github.com/jiva-studio/numen/libs/core/usecase/textversion/textcontract"
)

// Host finds the enabled plugins and starts them.
type Host interface {
	pluginport.PluginDiscoverer
	pluginport.PluginResolver
	pluginport.PluginActivator
	ListExtensions(mask domain.ContentType) []string
}

// Plugins is the enabled plugins that declare a text input, and their providers.
type Plugins struct {
	host Host
}

// NewPlugins reaches the text plugins of the host.
func NewPlugins(host Host) Plugins {
	return Plugins{host: host}
}

// GetTextPlugins is every enabled plugin that declares `texts`, in the order of their ids.
func (p Plugins) GetTextPlugins() []textport.TextPlugin {
	var out []textport.TextPlugin
	for _, info := range p.host.GetPluginInfos() {
		found, ok := p.host.GetPlugin(info.ID)
		if !ok || found.Manifest.Texts == nil {
			continue
		}
		out = append(out, textport.TextPlugin{
			ID: info.ID, Name: found.Manifest.Name, Files: p.getFileInput(found.Manifest.Texts.ContentTypes),
		})
	}
	return out
}

// getFileInput is the extensions, each with its dot, that give a file one of the types.
func (p Plugins) getFileInput(types []domain.ContentType) domain.FileInput {
	var out domain.FileInput
	for _, one := range types {
		for _, extension := range p.host.ListExtensions(one) {
			out = append(out, "."+extension)
		}
	}
	return out
}

// OpenProvider is the text provider of an enabled plugin that names content types under `texts`.
func (p Plugins) OpenProvider(_ context.Context, pluginID string) (textport.TextProvider, error) {
	found, ok := p.host.GetPlugin(pluginID)
	if !ok {
		return nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginDisabled}
	}
	if found.Manifest.Texts == nil || len(found.Manifest.Texts.ContentTypes) == 0 {
		return nil, fmt.Errorf("plugin %s makes no text of files on its own", pluginID)
	}
	return provider{service: bridge.Service[numenv1connect.TextProviderServiceClient]{
		Host: p.host, PluginID: pluginID, NewClient: numenv1connect.NewTextProviderServiceClient,
	}}, nil
}

// provider is one plugin's TextProviderService. Each call holds a reference on
// the plugin for as long as it runs.
type provider struct {
	service bridge.Service[numenv1connect.TextProviderServiceClient]
}

func (p provider) CreateText(ctx context.Context, req textport.TextCreation) error {
	client, lease, err := p.service.Open(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = lease.Close() }()
	res, err := client.CreateText(ctx, connect.NewRequest(&v1.TextProviderServiceCreateTextRequest{
		VaultId: string(req.Vault), Path: req.Path,
	}))
	if err != nil {
		return fmt.Errorf("rpc create text: %w", err)
	}
	return readError(res.Msg.GetError())
}

// readError is the error a provider answered with. A text that is not there is textport.ErrNoText.
func readError(code v1.TextErrorCode) error {
	switch code {
	case v1.TextErrorCode_TEXT_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.TextErrorCode_TEXT_ERROR_CODE_MISSING:
		return textport.ErrNoText
	case v1.TextErrorCode_TEXT_ERROR_CODE_NO_VAULT,
		v1.TextErrorCode_TEXT_ERROR_CODE_INVALID,
		v1.TextErrorCode_TEXT_ERROR_CODE_NOT_OWNER:
	}
	return fmt.Errorf("the text provider answered %s", code)
}

// NoPlugins is an installation with no plugin that makes text.
type NoPlugins struct{}

func (NoPlugins) GetTextPlugins() []textport.TextPlugin { return nil }

func (NoPlugins) OpenProvider(_ context.Context, pluginID string) (textport.TextProvider, error) {
	return nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginNotInstalled}
}
