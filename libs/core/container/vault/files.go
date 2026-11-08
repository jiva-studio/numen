package vault

import (
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
)

// OpenPluginFile builds the reader of a file in the store folder of a plugin.
func (c Config) OpenPluginFile() *file.OpenPluginFile {
	return file.NewOpenPluginFile(derivedfile.PluginStores{Options: c.vaultOptions()})
}

// GetMediaType builds the media type detector.
func (Config) GetMediaType() *file.GetMediaType {
	return file.NewGetMediaType(sharedmimeinfo.Open())
}
