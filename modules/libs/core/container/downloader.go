package container

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

var (
	errDownloaderUnavailable = errors.New("downloader plugin is unavailable or not installed")
)

var _ port.Downloader = (*hostDownloaderPort)(nil)
var _ port.Downloader = (*unavailableDownloader)(nil)

// DownloaderReady says whether a downloader plugin is active or present.
func (c Config) DownloaderReady() bool {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("downloader"); ok {
			return true
		}
		if _, ok := c.PluginHost.FindPluginByCapability("media.download"); ok {
			return true
		}
	}
	return false
}

// Downloader is what reaches an address on this machine.
func (c Config) Downloader(ctx context.Context) port.Downloader {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("downloader"); ok {
			return hostDownloaderPort{host: c.PluginHost}
		}
		if _, ok := c.PluginHost.FindPluginByCapability("media.download"); ok {
			return hostDownloaderPort{host: c.PluginHost}
		}
	}
	return unavailableDownloader{}
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
	if h.host == nil {
		return nil, errDownloaderUnavailable
	}
	return plugin.OpenDownloaderBridge(ctx, h.host)
}

func (h hostDownloaderPort) GetDownloadModel(at domain.URL) (port.DownloadModel, error) {
	bridge, err := h.openBridge(context.Background())
	if err != nil {
		return port.DownloadModel{}, err
	}
	defer bridge.Close()
	return bridge.GetDownloadModel(at)
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

type unavailableDownloader struct{}

func (unavailableDownloader) GetDownloadModel(_ domain.URL) (port.DownloadModel, error) {
	return port.DownloadModel{}, errDownloaderUnavailable
}

func (unavailableDownloader) GetMetadata(_ context.Context, _ domain.URL) (port.Metadata, error) {
	return port.Metadata{}, errDownloaderUnavailable
}

func (unavailableDownloader) GetText(_ context.Context, _ domain.URL, _ port.PreferredCaptions) (port.Text, error) {
	return port.Text{}, errDownloaderUnavailable
}

func (unavailableDownloader) Download(_ context.Context, _ domain.URL, _ io.Writer) (port.Copy, error) {
	return port.Copy{}, errDownloaderUnavailable
}
