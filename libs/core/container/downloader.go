package container

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/downloaderbridge"
	"github.com/jiva-studio/numen/libs/core/usecase/source"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/downloadport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

var _ downloadport.Downloader = (*hostDownloaderPort)(nil)
var _ downloadport.Downloader = (*unavailableDownloader)(nil)

// DownloaderReady says whether a downloader plugin is enabled.
func (c Config) DownloaderReady() bool {
	return c.hasPlugin(bridge.CapabilityDownloader)
}

// Downloader is what reaches an address on this machine.
func (c Config) Downloader(_ context.Context) downloadport.Downloader {
	host, err := c.resolvePlugin(bridge.CapabilityDownloader)
	if err != nil {
		return unavailableDownloader{why: err}
	}
	return hostDownloaderPort{host: host}
}

// ImportURL is one link note's address, downloaded into the vault, with the
// note cut again as soon as what came back is written.
func (c Config) ImportURL(ctx context.Context, db *Index, downloader downloadport.Downloader) source.ImportURL {
	level := c.Level(db)
	notes := c.Notes(db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), level)
	return source.ImportURL{
		Readers:     c.VaultReaders(),
		Derived:     c.GetDerivedStores(),
		By:          downloader,
		CopyMaxSize: c.Importing.CopyBytes(),
		IsToVault:   c.Importing.HasCopiesInVault(),
		Writers:     c.VaultWriters(),
		Languages:   c.Importing.Captions,
		IsAutomatic: c.Importing.CanUseAutomaticCaptions(),
		Cut: func(ctx context.Context, v domain.Vault, path string) error {
			return level(ctx, v, []string{path})
		},
		Names: func(ctx context.Context, v domain.Vault, path, title string) (string, error) {
			named, err := notes.Rename.Execute(ctx, v, path, title)
			return named.Path, err
		},
	}
}

type hostDownloaderPort struct {
	host pluginport.PluginManager
}

func (h hostDownloaderPort) openBridge(ctx context.Context) (*downloaderbridge.Downloader, error) {
	return downloaderbridge.Open(ctx, h.host)
}

func (h hostDownloaderPort) GetDownloadModel(ctx context.Context, targetURL domain.URL) (downloadport.DownloadModel, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return downloadport.DownloadModel{}, err
	}
	defer bridge.Close()
	return bridge.GetDownloadModel(ctx, targetURL)
}

func (h hostDownloaderPort) GetMetadata(ctx context.Context, rawURL domain.URL) (downloadport.Metadata, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return downloadport.Metadata{}, err
	}
	defer bridge.Close()
	return bridge.GetMetadata(ctx, rawURL)
}

func (h hostDownloaderPort) GetText(ctx context.Context, pos domain.URL, want downloadport.PreferredCaptions) (downloadport.Text, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return downloadport.Text{}, err
	}
	defer bridge.Close()
	return bridge.GetText(ctx, pos, want)
}

func (h hostDownloaderPort) Download(ctx context.Context, offset domain.URL, into io.Writer) (downloadport.Copy, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return downloadport.Copy{}, err
	}
	defer bridge.Close()
	return bridge.Download(ctx, offset, into)
}

// unavailableDownloader answers every ask with why no downloader plugin is there.
type unavailableDownloader struct {
	why error
}

func (u unavailableDownloader) GetDownloadModel(_ context.Context, _ domain.URL) (downloadport.DownloadModel, error) {
	return downloadport.DownloadModel{}, u.why
}

func (u unavailableDownloader) GetMetadata(_ context.Context, _ domain.URL) (downloadport.Metadata, error) {
	return downloadport.Metadata{}, u.why
}

func (u unavailableDownloader) GetText(_ context.Context, _ domain.URL, _ downloadport.PreferredCaptions) (downloadport.Text, error) {
	return downloadport.Text{}, u.why
}

func (u unavailableDownloader) Download(_ context.Context, _ domain.URL, _ io.Writer) (downloadport.Copy, error) {
	return downloadport.Copy{}, u.why
}
