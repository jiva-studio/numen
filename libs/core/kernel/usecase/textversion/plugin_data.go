package textversion

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// PluginData is the folders plugins keep in a vault's store, `.numen/<plugin id>/`,
// and the records of the text versions whose files they hold.
type PluginData struct {
	versions versionDeleter
	stores   pluginport.PluginStores
}

// versionDeleter deletes the records of the versions a plugin made.
type versionDeleter interface {
	DeletePluginVersions(ctx context.Context, v domain.Vault, pluginID string) error
}

// NewPluginData is the plugin folders opened through stores, with the text
// versions recorded in versions.
func NewPluginData(versions *Versions, stores pluginport.PluginStores) *PluginData {
	return &PluginData{versions: versions, stores: stores}
}

// GetPluginData is the size of the plugin's folder in the vault.
func (u *PluginData) GetPluginData(ctx context.Context, v domain.Vault, pluginID string) (plugin.DataSize, error) {
	store, err := u.stores.Open(v, pluginID)
	if err != nil {
		return plugin.DataSize{}, err
	}
	files, err := store.List(ctx)
	if err != nil {
		return plugin.DataSize{}, fmt.Errorf("list the folder of %s: %w", pluginID, err)
	}
	size := plugin.DataSize{FileCount: len(files)}
	for _, one := range files {
		size.ByteCount += one.Size
	}
	return size, nil
}

// DeletePluginData deletes the plugin's folder in the vault and the records of
// its text versions. The files whose versions changed are reported to the
// listener, which reindexes them.
func (u *PluginData) DeletePluginData(ctx context.Context, v domain.Vault, pluginID string) error {
	if err := u.stores.DeleteFolder(ctx, v, pluginID); err != nil {
		return fmt.Errorf("delete the folder of %s: %w", pluginID, err)
	}
	return u.versions.DeletePluginVersions(ctx, v, pluginID)
}
