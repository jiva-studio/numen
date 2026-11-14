package file

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// OpenPluginFile opens a file in a plugin's store of a vault, as a plugin:// address names it.
type OpenPluginFile struct {
	stores pluginport.PluginStores
}

func NewOpenPluginFile(stores pluginport.PluginStores) *OpenPluginFile {
	return &OpenPluginFile{stores: stores}
}

// PluginFile is a file of a plugin's store opened for partial reads. The caller closes it.
type PluginFile struct {
	Body io.ReadSeekCloser
	Size int64
}

// Execute opens the file at the path in the plugin's store of the vault.
func (u *OpenPluginFile) Execute(ctx context.Context, v domain.Vault, pluginID, path string) (PluginFile, error) {
	store, err := u.stores.Open(v, pluginID)
	if err != nil {
		return PluginFile{}, err
	}
	body, size, err := store.Open(ctx, path)
	if err != nil {
		return PluginFile{}, err
	}
	return PluginFile{Body: body, Size: size}, nil
}

type pluginFileOpener interface {
	Execute(ctx context.Context, v domain.Vault, pluginID, path string) (PluginFile, error)
}

// noPluginFiles is the opening of plugin files on an installation that keeps no plugin stores.
type noPluginFiles struct{}

func (noPluginFiles) Execute(context.Context, domain.Vault, string, string) (PluginFile, error) {
	return PluginFile{}, ErrNoPluginFiles
}

func getPluginFileOpener(open *OpenPluginFile) pluginFileOpener {
	if open == nil {
		return noPluginFiles{}
	}
	return open
}
