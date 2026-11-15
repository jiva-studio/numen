package pluginport

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// PluginStoreReader reads one plugin's own folder in a vault's store, `.numen/<plugin id>/`. A name is a file
// directly in that folder; one that holds a slash or begins with a dot gets vault.ErrOutside.
type PluginStoreReader interface {
	Read(ctx context.Context, name string) ([]byte, error)
	List(ctx context.Context) ([]plugin.StoreFile, error)
	Open(ctx context.Context, name string) (io.ReadSeekCloser, int64, error)
}

// PluginStores opens a plugin's folder in the store of a vault.
type PluginStores interface {
	Open(v domain.Vault, pluginID string) (PluginStoreReader, error)
	// DeleteFolder deletes the folder of the plugin and every file in it. A
	// folder that is not there is deleted.
	DeleteFolder(ctx context.Context, v domain.Vault, pluginID string) error
}
