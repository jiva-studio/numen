package downloaderbridge

import (
	"context"
	"errors"
	"fmt"
	"io"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/port/downloadport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// Host provides plugin resolution, activation, and permission checking for downloads.
type Host interface {
	pluginport.PluginResolver
	pluginport.PluginActivator
	pluginport.PluginAuthorizer
}

// Downloader connects port.Downloader calls to an out-of-process downloader plugin via Connect-RPC.
type Downloader struct {
	host     Host
	pluginID string
	service  bridge.Service[numenv1connect.DownloadServiceClient]
	lease    *bridge.Lease
}

// Open activates or connects to the downloader plugin and returns a port.Downloader.
func Open(ctx context.Context, host Host) (*Downloader, error) {
	plugin, err := host.ResolveCapability(bridge.CapabilityDownloader)
	if err != nil {
		return nil, err
	}

	_, lease, err := bridge.Open(ctx, host, plugin.Manifest.ID, numenv1connect.NewDownloadServiceClient)
	if err != nil {
		return nil, err
	}

	return &Downloader{
		host:     host,
		pluginID: plugin.Manifest.ID,
		service:  bridge.Service[numenv1connect.DownloadServiceClient]{Host: host, PluginID: plugin.Manifest.ID, NewClient: numenv1connect.NewDownloadServiceClient},
		lease:    lease,
	}, nil
}

// CanHandle reports whether the downloader plugin supports the given address.
func (b *Downloader) CanHandle(ctx context.Context, targetURL domain.URL) (bool, error) {
	req := connect.NewRequest(&v1.CanHandleRequest{
		Url: string(targetURL),
	})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.CanHandle(ctx, req)
	if err != nil {
		return false, fmt.Errorf("rpc can handle: %w", err)
	}
	return resp.Msg.GetCanHandle(), nil
}

// GetDownloadModel returns the tool identity and version for a given address.
func (b *Downloader) GetDownloadModel(ctx context.Context, rawURL domain.URL) (downloadport.DownloadModel, error) {
	ctx, cancel := context.WithTimeout(ctx, transport.ModelInfoTimeout)
	defer cancel()
	req := connect.NewRequest(&v1.GetDownloadModelRequest{
		Url: string(rawURL),
	})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return downloadport.DownloadModel{}, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.GetDownloadModel(ctx, req)
	if err != nil {
		return downloadport.DownloadModel{}, fmt.Errorf("rpc get download model: %w", err)
	}
	return downloadport.DownloadModel{
		Tool:     resp.Msg.GetTool(),
		Version:  resp.Msg.GetVersion(),
		Producer: resp.Msg.GetProducer(),
	}, nil
}

// GetMetadata returns media duration, language, and available caption tracks.
func (b *Downloader) GetMetadata(ctx context.Context, u domain.URL) (downloadport.Metadata, error) {
	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, plugindir.PermissionNetworkOutbound); err != nil {
			return downloadport.Metadata{}, err
		}
	}

	req := connect.NewRequest(&v1.GetMetadataRequest{
		Url: string(u),
	})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return downloadport.Metadata{}, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.GetMetadata(ctx, req)
	if err != nil {
		return downloadport.Metadata{}, fmt.Errorf("rpc get metadata: %w", err)
	}

	return downloadport.Metadata{
		Title:       resp.Msg.GetTitle(),
		Bytes:       resp.Msg.GetBytes(),
		Length:      int(resp.Msg.GetLength()),
		Language:    resp.Msg.GetLanguage(),
		Captions:    resp.Msg.GetCaptions(),
		IsAutomatic: resp.Msg.GetIsAutomatic(),
	}, nil
}

// GetText extracts transcripts, cues, or prose text from an address.
func (b *Downloader) GetText(ctx context.Context, pos domain.URL, want downloadport.PreferredCaptions) (downloadport.Text, error) {
	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, plugindir.PermissionNetworkOutbound); err != nil {
			return downloadport.Text{}, err
		}
	}

	req := connect.NewRequest(&v1.GetTextRequest{
		Url: string(pos),
		PreferredCaptions: &v1.PreferredCaptions{
			Languages:   want.Languages,
			IsAutomatic: want.IsAutomatic,
		},
	})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return downloadport.Text{}, err
	}
	defer func() { _ = call.Close() }()

	resp, err := client.GetText(ctx, req)
	if err != nil {
		var connErr *connect.Error
		if errors.As(err, &connErr) && connErr.Code() == connect.CodeNotFound {
			return downloadport.Text{}, downloadport.ErrNothingDownloaded
		}
		return downloadport.Text{}, fmt.Errorf("rpc get text: %w", err)
	}

	cues := make([]domain.TranscriptCue, len(resp.Msg.GetCues()))
	for i, c := range resp.Msg.GetCues() {
		cues[i] = domain.TranscriptCue{
			Text: c.GetText(),
			From: int(c.GetFrom()),
			To:   int(c.GetTo()),
		}
	}

	return downloadport.Text{
		Producer: resp.Msg.GetProducer(),
		Cues:     cues,
		Prose:    resp.Msg.GetProse(),
		Title:    resp.Msg.GetTitle(),
		Length:   int(resp.Msg.GetLength()),
	}, nil
}

// Download streams media or page content from the address into the given destination writer.
func (b *Downloader) Download(ctx context.Context, offset domain.URL, into io.Writer) (downloadport.Copy, error) {
	if b.host != nil {
		if err := b.host.CheckPluginPermission(b.pluginID, plugindir.PermissionNetworkOutbound); err != nil {
			return downloadport.Copy{}, err
		}
	}

	req := connect.NewRequest(&v1.DownloadMediaRequest{
		Url: string(offset),
	})
	client, call, err := b.service.Open(ctx)
	if err != nil {
		return downloadport.Copy{}, err
	}
	defer func() { _ = call.Close() }()

	stream, err := client.DownloadMedia(ctx, req)
	if err != nil {
		var connErr *connect.Error
		if errors.As(err, &connErr) && connErr.Code() == connect.CodeNotFound {
			return downloadport.Copy{}, downloadport.ErrNothingDownloaded
		}
		return downloadport.Copy{}, fmt.Errorf("rpc download media: %w", err)
	}

	return consumeMediaStream(stream, into)
}

func consumeMediaStream(stream *connect.ServerStreamForClient[v1.DownloadMediaResponse], into io.Writer) (downloadport.Copy, error) {
	var copyInfo downloadport.Copy
	for stream.Receive() {
		msg := stream.Msg()
		if len(msg.GetChunk()) > 0 {
			if _, err := into.Write(msg.GetChunk()); err != nil {
				_ = stream.Close()
				return downloadport.Copy{}, fmt.Errorf("write media chunk: %w", err)
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
			return downloadport.Copy{}, downloadport.ErrNothingDownloaded
		}
		return downloadport.Copy{}, fmt.Errorf("download media stream: %w", err)
	}

	return copyInfo, nil
}

// Close releases the active reference count for the downloader plugin in the host supervisor.
func (b *Downloader) Close() error {
	return b.lease.Close()
}
