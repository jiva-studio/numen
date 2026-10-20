package container

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

var _ port.Downloader = (*hostDownloaderPort)(nil)
var _ port.Downloader = (*unavailableDownloader)(nil)

// DownloaderReady says whether a downloader plugin is enabled.
func (c Config) DownloaderReady() bool {
	return c.hasPlugin(plugin.CapabilityDownloader)
}

// Downloader is what reaches an address on this machine.
func (c Config) Downloader(_ context.Context) port.Downloader {
	host, err := c.resolvePlugin(plugin.CapabilityDownloader)
	if err != nil {
		return unavailableDownloader{why: err}
	}
	return hostDownloaderPort{host: host}
}

// ImportURL is one link note's address, downloaded into the vault, with the
// note cut again as soon as what came back is written. A machine holding
// neither tool has none.
func (c Config) ImportURL(ctx context.Context, db *Index, by port.Downloader) source.ImportURL {
	level := c.Level(db)
	notes := c.Notes(db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), level)
	return source.ImportURL{
		Readers:     c.VaultReaders(),
		Derived:     c.GetDerivedStores(),
		By:          by,
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
	host port.PluginManager
}

func (h hostDownloaderPort) openBridge(ctx context.Context) (*plugin.DownloaderBridge, error) {
	return plugin.OpenDownloaderBridge(ctx, h.host)
}

func (h hostDownloaderPort) GetDownloadModel(ctx context.Context, at domain.URL) (port.DownloadModel, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return port.DownloadModel{}, err
	}
	defer bridge.Close()
	return bridge.GetDownloadModel(ctx, at)
}

func (h hostDownloaderPort) GetMetadata(ctx context.Context, at domain.URL) (port.Metadata, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return port.Metadata{}, err
	}
	defer bridge.Close()
	return bridge.GetMetadata(ctx, at)
}

func (h hostDownloaderPort) GetText(ctx context.Context, at domain.URL, want port.PreferredCaptions) (port.Text, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return port.Text{}, err
	}
	defer bridge.Close()
	return bridge.GetText(ctx, at, want)
}

func (h hostDownloaderPort) Download(ctx context.Context, at domain.URL, into io.Writer) (port.Copy, error) {
	bridge, err := h.openBridge(ctx)
	if err != nil {
		return port.Copy{}, err
	}
	defer bridge.Close()
	return bridge.Download(ctx, at, into)
}

// unavailableDownloader answers every ask with why no downloader plugin is there.
type unavailableDownloader struct {
	why error
}

func (u unavailableDownloader) GetDownloadModel(_ context.Context, _ domain.URL) (port.DownloadModel, error) {
	return port.DownloadModel{}, u.why
}

func (u unavailableDownloader) GetMetadata(_ context.Context, _ domain.URL) (port.Metadata, error) {
	return port.Metadata{}, u.why
}

func (u unavailableDownloader) GetText(_ context.Context, _ domain.URL, _ port.PreferredCaptions) (port.Text, error) {
	return port.Text{}, u.why
}

func (u unavailableDownloader) Download(_ context.Context, _ domain.URL, _ io.Writer) (port.Copy, error) {
	return port.Copy{}, u.why
}
