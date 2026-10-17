package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errNoDownloaderPlugin = errors.New("no active media downloader plugin found")
)

// DownloaderPluginHost provides plugin resolution, activation, and permission checking for downloads.
type DownloaderPluginHost interface {
	port.PluginResolver
	port.PluginActivator
	port.PluginAuthorizer
}

// DownloaderBridge connects port.Downloader calls to an out-of-process downloader plugin via Connect-RPC.
type DownloaderBridge struct {
	host     DownloaderPluginHost
	pluginID string
	client   numenv1connect.DownloadServiceClient
}

// OpenDownloaderBridge activates or connects to the downloader plugin and returns a port.Downloader.
func OpenDownloaderBridge(ctx context.Context, host DownloaderPluginHost) (*DownloaderBridge, error) {
	plugin, ok := host.FindPluginByCapability("downloader")
	if !ok {
		plugin, ok = host.FindPluginByCapability("media.download")
	}
	if !ok {
		return nil, errNoDownloaderPlugin
	}

	endpoint, err := host.EnsureActive(ctx, plugin.Manifest.ID)
	if err != nil {
		return nil, fmt.Errorf("ensure active %s: %w", plugin.Manifest.ID, err)
	}

	httpClient := EndpointClient(endpoint)
	interceptor := connect.WithInterceptors(authInterceptor(endpoint.Token))
	client := numenv1connect.NewDownloadServiceClient(httpClient, endpoint.BaseURL(), interceptor)

	return &DownloaderBridge{
		host:     host,
		pluginID: plugin.Manifest.ID,
		client:   client,
	}, nil
}

// CanHandle reports whether the downloader plugin supports the given address.
func (b *DownloaderBridge) CanHandle(ctx context.Context, at domain.URL) (bool, error) {
	req := connect.NewRequest(&v1.CanHandleRequest{
		Url: string(at),
	})
	resp, err := b.client.CanHandle(ctx, req)
	if err != nil {
		return false, fmt.Errorf("rpc can handle: %w", err)
	}
	return resp.Msg.GetCanHandle(), nil
}

// GetDownloadModel returns the tool identity and version for a given address.
func (b *DownloaderBridge) GetDownloadModel(ctx context.Context, at domain.URL) (port.DownloadModel, error) {
	ctx, cancel := context.WithTimeout(ctx, modelInfoTimeout)
	defer cancel()
	req := connect.NewRequest(&v1.GetDownloadModelRequest{
		Url: string(at),
	})
	resp, err := b.client.GetDownloadModel(ctx, req)
	if err != nil {
		return port.DownloadModel{}, fmt.Errorf("rpc get download model: %w", err)
	}
	return port.DownloadModel{
		Tool:     resp.Msg.GetTool(),
		Version:  resp.Msg.GetVersion(),
		Producer: resp.Msg.GetProducer(),
	}, nil
}

// GetMetadata returns media duration, language, and available caption tracks.
func (b *DownloaderBridge) GetMetadata(ctx context.Context, at domain.URL) (port.Metadata, error) {
	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, PermissionNetworkOutbound); err != nil {
			return port.Metadata{}, err
		}
	}

	req := connect.NewRequest(&v1.GetMetadataRequest{
		Url: string(at),
	})
	resp, err := b.client.GetMetadata(ctx, req)
	if err != nil {
		return port.Metadata{}, fmt.Errorf("rpc get metadata: %w", err)
	}

	return port.Metadata{
		Title:       resp.Msg.GetTitle(),
		Bytes:       resp.Msg.GetBytes(),
		Length:      int(resp.Msg.GetLength()),
		Language:    resp.Msg.GetLanguage(),
		Captions:    resp.Msg.GetCaptions(),
		IsAutomatic: resp.Msg.GetIsAutomatic(),
	}, nil
}

// GetText extracts transcripts, cues, or prose text from an address.
func (b *DownloaderBridge) GetText(ctx context.Context, at domain.URL, want port.PreferredCaptions) (port.Text, error) {
	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, PermissionNetworkOutbound); err != nil {
			return port.Text{}, err
		}
	}

	req := connect.NewRequest(&v1.GetTextRequest{
		Url: string(at),
		PreferredCaptions: &v1.PreferredCaptions{
			Languages:   want.Languages,
			IsAutomatic: want.IsAutomatic,
		},
	})
	resp, err := b.client.GetText(ctx, req)
	if err != nil {
		var connErr *connect.Error
		if errors.As(err, &connErr) && connErr.Code() == connect.CodeNotFound {
			return port.Text{}, port.ErrNothingDownloaded
		}
		return port.Text{}, fmt.Errorf("rpc get text: %w", err)
	}

	cues := make([]domain.TranscriptCue, len(resp.Msg.GetCues()))
	for i, c := range resp.Msg.GetCues() {
		cues[i] = domain.TranscriptCue{
			Text: c.GetText(),
			From: int(c.GetFrom()),
			To:   int(c.GetTo()),
		}
	}

	return port.Text{
		Producer: resp.Msg.GetProducer(),
		Cues:     cues,
		Prose:    resp.Msg.GetProse(),
		Title:    resp.Msg.GetTitle(),
		Length:   int(resp.Msg.GetLength()),
	}, nil
}

// Download streams media or page content from the address into the given destination writer.
func (b *DownloaderBridge) Download(ctx context.Context, at domain.URL, into io.Writer) (port.Copy, error) {
	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, PermissionNetworkOutbound); err != nil {
			return port.Copy{}, err
		}
	}

	req := connect.NewRequest(&v1.DownloadMediaRequest{
		Url: string(at),
	})
	stream, err := b.client.DownloadMedia(ctx, req)
	if err != nil {
		var connErr *connect.Error
		if errors.As(err, &connErr) && connErr.Code() == connect.CodeNotFound {
			return port.Copy{}, port.ErrNothingDownloaded
		}
		return port.Copy{}, fmt.Errorf("rpc download media: %w", err)
	}

	return consumeMediaStream(stream, into)
}

func consumeMediaStream(stream *connect.ServerStreamForClient[v1.DownloadMediaResponse], into io.Writer) (port.Copy, error) {
	var copyInfo port.Copy
	for stream.Receive() {
		msg := stream.Msg()
		if len(msg.GetChunk()) > 0 {
			if _, err := into.Write(msg.GetChunk()); err != nil {
				_ = stream.Close()
				return port.Copy{}, fmt.Errorf("write media chunk: %w", err)
			}
		}
		if msg.GetMediaType() != "" {
			copyInfo.MediaType = msg.GetMediaType()
		}
		if msg.GetExtension() != "" {
			copyInfo.Extension = msg.GetExtension()
		}
	}
	if err := stream.Err(); err != nil {
		var connErr *connect.Error
		if errors.As(err, &connErr) && connErr.Code() == connect.CodeNotFound {
			return port.Copy{}, port.ErrNothingDownloaded
		}
		return port.Copy{}, fmt.Errorf("download media stream: %w", err)
	}

	return copyInfo, nil
}

// Close releases the active reference count for the downloader plugin in the host supervisor.
func (b *DownloaderBridge) Close() error {
	b.host.Release(b.pluginID)
	return nil
}
