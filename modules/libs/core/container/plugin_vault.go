package container

import (
	"context"
	"fmt"
	"io/fs"
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
	if c.PluginCachePath != "" {
		return appstate.PluginCachesAt(filepath.Join(c.PluginCachePath, "plugins")), nil
	}
	kept, err := appstate.OpenPluginCaches()
	if err != nil {
		return nil, err
	}
	return kept, nil
}

// PluginVaultHooks is what a window adds to the vault it lets the plugins work.
type PluginVaultHooks struct {
	// Written is told after a plugin changed the vault.
	Written func()
	// IndexState starts reading a vault into the index when it is not there and says where that
	// stands. Left nil, a vault the index carries is indexed and any other is being indexed.
	IndexState func(ctx context.Context, v domain.Vault) (port.PluginIndexState, error)
	// InUse is told that a plugin is working a vault for the person.
	InUse func(ctx context.Context, v domain.Vault)
	// Moved tells that something moved in some vault. Left nil, it is the index file changing.
	Moved func(ctx context.Context) (<-chan struct{}, error)
}

// OpenPluginVault is the vault the plugins work through the host: the one current names, over the
// files, the index and the stores of this installation.
func (e EditorAssembly) OpenPluginVault(
	current func() (domain.Vault, bool), hooks PluginVaultHooks,
) (port.PluginVault, error) {
	return e.cfg.OpenPluginVault(e.registry, e.db, current, hooks)
}

// OpenPluginVault is the vault the plugins work through the host over the vaults of a list and the
// index that holds them: the one current names, else the one a call names by identity.
func (c Config) OpenPluginVault(
	registry port.VaultRegistry, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) (port.PluginVault, error) {
	cache, err := c.OpenPluginCache()
	if err != nil {
		return nil, err
	}
	refresh := makeRefresh(c, db, db.Notes())
	queries := db.Queries()
	vaults := plugin.Vault{
		Current:  current,
		Readers:  c.VaultReaders(),
		Writers:  c.VaultWriters(),
		Notes:    queries,
		Links:    db.Links(),
		Problems: db.Problems(),
		Data:     filesystem.PluginData{Options: c.vaultOptions()},
		Cache:    cache,
		Watcher:  c.VaultWatcher(),
		Written:  hooks.Written,
		Vaults:   registry.List,
		Find: func(id domain.VaultID) (domain.Vault, error) {
			found, isFound, err := registry.FindVault(string(id))
			if err != nil {
				return domain.Vault{}, err
			}
			if !isFound {
				return domain.Vault{}, fmt.Errorf("vault %q: %w", id, fs.ErrNotExist)
			}
			return found, nil
		},
		InUse:      hooks.InUse,
		IndexState: hooks.IndexState,
		Moved:      hooks.Moved,
		Index: func(ctx context.Context, v domain.Vault, paths []string) error {
			_, err := refresh.Execute(ctx, v, paths)
			return err
		},
	}
	if vaults.IndexState == nil {
		vaults.IndexState = func(ctx context.Context, v domain.Vault) (port.PluginIndexState, error) {
			has, err := queries.Has(ctx, v.ID)
			if err != nil {
				return port.PluginIndexState{}, err
			}
			if has {
				return port.PluginIndexState{Status: port.PluginIndexed}, nil
			}
			return port.PluginIndexState{Status: port.PluginIndexing}, nil
		}
	}
	if vaults.Moved == nil {
		vaults.Moved = c.Moves
	}
	return vaults, nil
}
