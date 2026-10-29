package container

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hostvault"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/servicebridge"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// pluginGracePeriod is how long a plugin has to exit after a stop request.
const pluginGracePeriod = 2 * time.Second

// errSafeMode is an installation started in safe mode, which starts no plugin.
var errSafeMode = errors.New("numen is running in safe mode")

const pluginFailureTaskPrefix = "plugin-failed:"

// resolvePlugin is the plugin host where an enabled plugin declares the capability. Where none
// does, the error is a *port.UnavailableError: an installation with no plugin host has none
// installed, and one in safe mode has every one disabled.
func (c Config) resolvePlugin(capability string) (pluginport.PluginManager, error) {
	if c.PluginHost == nil {
		return nil, &pluginport.UnavailableError{Capability: capability, Reason: pluginport.PluginNotInstalled}
	}
	if c.SafeMode {
		return nil, &pluginport.UnavailableError{Capability: capability, Reason: pluginport.PluginDisabled, Err: errSafeMode}
	}
	if _, err := c.PluginHost.ResolveCapability(capability); err != nil {
		return nil, err
	}
	return c.PluginHost, nil
}

// hasPlugin is whether an enabled plugin declares the capability.
func (c Config) hasPlugin(capability string) bool {
	_, err := c.resolvePlugin(capability)
	return err == nil
}

// SetSafeMode is the configuration started in safe mode or out of it, and its plugin host with it.
// In safe mode the host stops every plugin it runs and starts none.
func (c Config) SetSafeMode(enabled bool) Config {
	c.SafeMode = enabled
	if c.PluginHost != nil {
		c.PluginHost.SetSafeMode(enabled)
	}
	return c
}

// SetPluginVaultPath is the vault the plugins' paths are inside.
func (c Config) SetPluginVaultPath(path string) {
	if c.PluginHost != nil {
		c.PluginHost.SetVaultPath(path)
	}
}

// StopPlugins stops every plugin the host runs.
func (c Config) StopPlugins(ctx context.Context) error {
	if c.PluginHost == nil {
		return nil
	}
	return c.PluginHost.TeardownAll(ctx, pluginGracePeriod)
}

// GetPluginCommands is the use cases over the commands the plugins declare. Where there is no plugin
// host they list none.
func (c Config) GetPluginCommands() plugincommand.Commands {
	if c.PluginHost == nil {
		return plugincommand.NewCommands(nil, nil, nil)
	}
	return plugincommand.NewCommands(commandbridge.Open(c.PluginHost), c.PluginHost, c.VaultReaders())
}

// OpenPluginCommands opens the index of the vault and lets the plugins work it, its text versions
// and the queue through the host with no window, and answers how to run a command of theirs and
// how to close what it opened.
func (c Config) OpenPluginCommands(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return plugincommand.Run{}, nil, err
	}
	if _, err := c.OpenPluginWork(ctx, db, v); err != nil {
		_ = db.Close()
		return plugincommand.Run{}, nil, err
	}
	return c.GetPluginCommands().Run, db.Close, nil
}

// OpenPluginWork lets the plugins work one vault, its text versions and the queue through the host
// with no window, over the index given. The configuration it returns reads the text of a file
// through the text versions, as a window does.
func (c Config) OpenPluginWork(ctx context.Context, db *Index, v domain.Vault) (Config, error) {
	c.SetPluginVaultPath(v.Path)
	versions, err := c.NewTextVersions(db)
	if err != nil {
		return c, err
	}
	c.TextVersions = versions
	registry, err := c.Registry()
	if err != nil {
		return c, err
	}
	current := func() (domain.Vault, bool) { return v, true }
	vault, err := c.OpenPluginVault(registry, db, current, PluginVaultHooks{})
	if err != nil {
		return c, err
	}
	c.SetPluginWork(run.NewTasks(), &run.Lock{})
	c.SetPluginVault(vault)
	c.SetPluginTexts()
	return c, nil
}

// PluginServices is where a window forwards the calls it receives for the services plugins serve.
type PluginServices = interface {
	http.Handler
	// Serves reports whether an enabled plugin serves the service the path names.
	Serves(urlPath string) bool
	// ServeRoute forwards a request on the plugin route, /plugin/<id>/..., to that plugin.
	ServeRoute(w http.ResponseWriter, r *http.Request)
	// ListFrameOrigins lists the origins the pages of every enabled plugin frame.
	ListFrameOrigins() []string
	// GetFrameOrigins lists the origins the pages of one enabled plugin frame.
	GetFrameOrigins(pluginID string) []string
}

// GetPluginServices is the forwarding of the services plugins serve to the windows. Where there is
// no plugin host it serves none.
func (c Config) GetPluginServices() PluginServices {
	if c.PluginHost == nil {
		return servicebridge.NoServices{}
	}
	return servicebridge.Open(c.PluginHost)
}

// GetPluginRetrier restarts, on request from the window, a plugin the host has suspended.
func (c Config) GetPluginRetrier() func(ctx context.Context, pluginID string) (*pluginport.PluginFailure, error) {
	return func(ctx context.Context, pluginID string) (*pluginport.PluginFailure, error) {
		if c.PluginHost == nil {
			return nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginNotInstalled}
		}
		return c.PluginHost.RetryPlugin(ctx, pluginID)
	}
}

// ReportPluginFailures keeps one task in the list for each plugin the host suspends, with the reason as its
// error, and takes it out when the host forgets the failure.
func (c Config) ReportPluginFailures(tasks *run.Tasks) {
	if c.PluginHost == nil {
		return
	}
	c.PluginHost.SetOnPluginFailed(func(failure pluginport.PluginFailure) {
		tasks.Set(run.Task{
			ID:         pluginFailureTaskPrefix + failure.PluginID,
			Title:      "Plugin stopped",
			About:      failure.Name,
			Error:      failure.Reason,
			PluginID:   failure.PluginID,
			PluginName: failure.Name,
		})
	})
	c.PluginHost.SetOnFailureCleared(func(pluginID string) {
		tasks.Remove(pluginFailureTaskPrefix + pluginID)
	})
}

// SetPluginWork opens the lock the models are held under and the list of tasks to the plugins.
func (c Config) SetPluginWork(tasks *run.Tasks, models *run.Lock) {
	if c.PluginHost != nil {
		c.PluginHost.SetPluginWork(run.NewPluginWork(models, tasks, c.getPluginName))
	}
}

// getPluginName is the name given by the plugin's manifest, or its id when the host has no such plugin.
func (c Config) getPluginName(pluginID string) string {
	if found, isKnown := c.PluginHost.GetPlugin(pluginID); isKnown && found.Manifest.Name != "" {
		return found.Manifest.Name
	}
	return pluginID
}

// SetPluginVault is the vault the plugins use through the host. Without one, a plugin that requires
// the vault receives an error stating that the host provides none.
func (c Config) SetPluginVault(vault pluginport.PluginVault) {
	if c.PluginHost != nil {
		c.PluginHost.SetPluginVault(vault)
	}
}

// OpenPluginCache is where a plugin keeps what it worked out for a vault: the folder the
// configuration names for schedules, else the platform's cache location.
func (c Config) OpenPluginCache() (pluginport.PluginCache, error) {
	if c.PluginCachePath != "" {
		return appstate.PluginCachesAt(filepath.Join(c.PluginCachePath, "plugins")), nil
	}
	retained, err := appstate.OpenPluginCaches()
	if err != nil {
		return nil, err
	}
	return retained, nil
}

// PluginVaultHooks is what a window adds to the vault it lets the plugins work.
type PluginVaultHooks struct {
	// Written is called after a plugin changes the vault.
	Written func()
	// IndexState starts reading a vault into the index when it is not indexed and reports its state. When
	// nil, a vault the index holds is treated as indexed and any other as being indexed.
	IndexState func(ctx context.Context, v domain.Vault) (pluginport.PluginIndexState, error)
	// Moved signals that something moved in some vault. When nil, the signal is the index file changing.
	Moved func(ctx context.Context) (<-chan struct{}, error)
}

// OpenPluginVault is the vault the plugins work through the host over the vaults of a list and the
// index that holds them: the one current names, else the one a call names by identity.
func (c Config) OpenPluginVault(
	registry vaultport.VaultRegistry, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) (pluginport.PluginVault, error) {
	cache, err := c.OpenPluginCache()
	if err != nil {
		return nil, err
	}
	refresh := makeRefresh(c, db, db.Notes())
	queries := db.Queries()
	vaults := hostvault.Vault{
		Current:  current,
		Readers:  c.VaultReaders(),
		Writers:  c.VaultWriters(),
		Notes:    queries,
		Links:    db.Links(),
		Problems: db.Problems(),
		Stores:   derivedfile.PluginStores{Options: c.vaultOptions()},
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
		IndexState: hooks.IndexState,
		Moved:      hooks.Moved,
		Index: func(ctx context.Context, v domain.Vault, paths []string) error {
			_, err := refresh.Execute(ctx, v, paths)
			return err
		},
	}
	if vaults.IndexState == nil {
		vaults.IndexState = func(ctx context.Context, v domain.Vault) (pluginport.PluginIndexState, error) {
			has, err := queries.Has(ctx, v.ID)
			if err != nil {
				return pluginport.PluginIndexState{}, err
			}
			if has {
				return pluginport.PluginIndexState{Status: pluginport.PluginIndexed}, nil
			}
			return pluginport.PluginIndexState{Status: pluginport.PluginIndexing}, nil
		}
	}
	if vaults.Moved == nil {
		vaults.Moved = c.Moves
	}
	return vaults, nil
}

var errNoPluginHost = errors.New("this installation has no plugin host to keep a secret for")

// GetPluginSecretWriter stores, on request from the window, the key of a secret a plugin
// declares in its manifest, and passes the plugin the secrets it currently holds.
func (c Config) GetPluginSecretWriter() func(ctx context.Context, pluginID, name, key string) error {
	return func(ctx context.Context, pluginID, name, key string) error {
		if c.PluginHost == nil {
			return errNoPluginHost
		}
		return c.PluginHost.SetPluginSecret(ctx, pluginID, name, key)
	}
}

// GetPluginSecretDeleter removes, on request from the window, the key of a secret a
// plugin declares in its manifest, and passes the plugin the secrets it currently holds.
func (c Config) GetPluginSecretDeleter() func(ctx context.Context, pluginID, name string) error {
	return func(ctx context.Context, pluginID, name string) error {
		if c.PluginHost == nil {
			return errNoPluginHost
		}
		return c.PluginHost.DeletePluginSecret(ctx, pluginID, name)
	}
}

// GetPluginDir is the folder of an installed plugin, and false for one not installed.
func (c Config) GetPluginDir(id string) (string, bool) {
	if c.PluginHost == nil {
		return "", false
	}
	p, ok := c.PluginHost.GetPlugin(id)
	if !ok {
		return "", false
	}
	return p.Dir, true
}

// GetFrontendPluginIDs are the plugins of the host's target that declare a frontend, in the order of
// their ids.
func (c Config) GetFrontendPluginIDs() []string {
	if c.PluginHost == nil {
		return nil
	}
	infos := c.PluginHost.GetPluginInfos()
	ids := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.HasFrontend {
			continue
		}
		ids = append(ids, info.ID)
	}
	return ids
}
