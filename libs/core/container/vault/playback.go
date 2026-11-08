package vault

import (
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/playback"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/port/playbackport"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
)

// Copies are the copies the enabled plugins keep of the vault's files in their store folders. An
// installation with no plugins keeps none.
func (c Config) Copies() playbackport.Copies {
	if c.PluginHost == nil || c.SafeMode {
		return playback.NoProxies{}
	}
	return playback.NewProxies(c.PluginHost, derivedfile.PluginStores{Options: c.vaultOptions()})
}

// OpenProxy builds the file proxy reader over vault readers and plugin copies.
func (c Config) OpenProxy() *file.OpenProxy {
	return file.NewOpenProxy(c.VaultReaders(), c.Copies())
}

// GetMediaType builds the media type detector.
func (Config) GetMediaType() *file.GetMediaType {
	return file.NewGetMediaType(sharedmimeinfo.Open())
}
