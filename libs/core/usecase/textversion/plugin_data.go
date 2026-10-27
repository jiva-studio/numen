package textversion

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// PluginData is the folders plugins keep in a vault's store, `.numen/<plugin id>/`,
// and the records of the text versions whose files they hold.
type PluginData struct {
	versions *Versions
	stores   pluginport.PluginStores
}

// NewPluginData is the plugin folders opened through stores, with the text
// versions recorded in versions.
func NewPluginData(versions *Versions, stores pluginport.PluginStores) (*PluginData, error) {
	if versions == nil || stores == nil {
		return nil, errors.New("plugin data needs text versions and plugin stores")
	}
	return &PluginData{versions: versions, stores: stores}, nil
}

// PluginDataSize is how many files a plugin's folder holds, and their bytes.
type PluginDataSize struct {
	FileCount int
	ByteCount int64
}

// GetPluginData is the size of the plugin's folder in the vault.
func (u *PluginData) GetPluginData(ctx context.Context, v domain.Vault, pluginID string) (PluginDataSize, error) {
	store, err := u.stores.Open(v, pluginID)
	if err != nil {
		return PluginDataSize{}, err
	}
	files, err := store.List(ctx)
	if err != nil {
		return PluginDataSize{}, fmt.Errorf("list the folder of %s: %w", pluginID, err)
	}
	size := PluginDataSize{FileCount: len(files)}
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
