package container

import (
	"context"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// SetPluginVault is the vault the plugins work through the host. Without one a plugin that calls for
// the vault is told the host shows none.
func (c Config) SetPluginVault(vault port.PluginVault) {
	if c.PluginHost != nil {
		c.PluginHost.SetPluginVault(vault)
	}
}

// OpenPluginCache is where a plugin keeps what it worked out for a vault: the folder the
// configuration names for schedules, else the platform's cache location.
func (c Config) OpenPluginCache() (port.PluginCache, error) {
	if c.SchedulesPath != "" {
		return appstate.PluginCachesAt(filepath.Join(c.SchedulesPath, "plugins")), nil
	}
	kept, err := appstate.OpenPluginCaches()
	if err != nil {
		return nil, err
	}
	return kept, nil
}

// OpenPluginVault is the vault the plugins work through the host: the one current names, over the
// files, the index and the stores of this installation.
func (e EditorAssembly) OpenPluginVault(current func() (domain.Vault, bool)) (port.PluginVault, error) {
	cache, err := e.cfg.OpenPluginCache()
	if err != nil {
		return nil, err
	}
	refresh := e.GetRefresh()
	return plugin.Vault{
		Current:  current,
		Readers:  e.cfg.VaultReaders(),
		Writers:  e.cfg.VaultWriters(),
		Notes:    e.db.Queries(),
		Links:    e.db.Links(),
		Problems: e.db.Problems(),
		Data:     filesystem.PluginData{Options: e.cfg.vaultOptions()},
		Cache:    cache,
		Watcher:  e.cfg.VaultWatcher(),
		Index: func(ctx context.Context, v domain.Vault, paths []string) error {
			_, err := refresh.Execute(ctx, v, paths)
			return err
		},
	}, nil
}
