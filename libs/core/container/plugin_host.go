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
	"github.com/jiva-studio/numen/libs/core/internal/adapter/keychain"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/serviceproxy"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/toolsbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/vaultaccess"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/usecase/plugintool"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// pluginGracePeriod is how long a plugin has to exit once it is told to stop.
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

// GetPluginTools is the use cases that offer an agent the tools the plugins serve. Where there is
// no plugin host they offer none.
func (c Config) GetPluginTools() plugintool.Tools {
	var tools pluginport.PluginTools
	if c.PluginHost != nil {
		tools = toolsbridge.Open(c.PluginHost)
	}
	return plugintool.NewTools(tools)
}

// GetPluginCommands is the use cases over the commands the plugins declare. Where there is no plugin
// host they list none.
func (c Config) GetPluginCommands() plugincommand.Commands {
	if c.PluginHost == nil {
		return plugincommand.NewCommands(nil)
	}
	return plugincommand.NewCommands(commandbridge.Open(c.PluginHost))
}

// PluginServices is where a window forwards the calls it receives for the services plugins serve.
type PluginServices = interface {
	http.Handler
	// Serves reports whether an enabled plugin serves the service the path names.
	Serves(urlPath string) bool
}

// GetPluginServices is the forwarding of the services plugins serve to the windows. Where there is
// no plugin host it serves none.
func (c Config) GetPluginServices() PluginServices {
	if c.PluginHost == nil {
		return noPluginServices{}
	}
	return serviceproxy.Open(c.PluginHost)
}

// noPluginServices is the forwarding of an installation with no plugin host.
type noPluginServices struct{}

func (noPluginServices) Serves(string) bool { return false }

func (noPluginServices) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "no plugin serves this service", http.StatusNotFound)
}

// GetPluginRetrier starts, as the window asks, a plugin the host suspends once more.
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
			ID:    pluginFailureTaskPrefix + failure.PluginID,
			Title: "Plugin stopped",
			About: failure.Name,
			Error: failure.Reason,
		})
	})
	c.PluginHost.SetOnFailureCleared(func(pluginID string) {
		tasks.Remove(pluginFailureTaskPrefix + pluginID)
	})
}

// SetPluginVault is the vault the plugins work through the host. Without one a plugin that calls for
// the vault is told the host shows none.
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
	// Written is told after a plugin changed the vault.
	Written func()
	// IndexState starts reading a vault into the index when it is not there and says where that
	// stands. Left nil, a vault the index carries is indexed and any other is being indexed.
	IndexState func(ctx context.Context, v domain.Vault) (pluginport.PluginIndexState, error)
	// InUse is told that a plugin is working a vault for the person.
	InUse func(ctx context.Context, v domain.Vault)
	// Moved tells that something moved in some vault. Left nil, it is the index file changing.
	Moved func(ctx context.Context) (<-chan struct{}, error)
}

// OpenPluginVault is the vault the plugins work through the host: the one current names, over the
// files, the index and the stores of this installation.
func (e EditorAssembly) OpenPluginVault(
	current func() (domain.Vault, bool), hooks PluginVaultHooks,
) (pluginport.PluginVault, error) {
	return e.cfg.OpenPluginVault(e.registry, e.db, current, hooks)
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
	vaults := vaultaccess.Vault{
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
		InUse:      hooks.InUse,
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

// GetKeyStore is where the keys of services are kept: the one the installation
// names, else the system keychain. A machine without a keychain fails with
// port.ErrNoKeyStore and keeps no key anywhere else.
func (c Config) GetKeyStore() pluginport.KeyStore {
	if c.KeyStore != nil {
		return c.KeyStore
	}
	return keychain.Store{}
}

// GetPluginSecretWriter stores, as the window asks, the key of a secret a plugin
// declares in its manifest, and hands the plugin the secrets it holds now.
func (c Config) GetPluginSecretWriter() func(ctx context.Context, pluginID, name, key string) error {
	return func(ctx context.Context, pluginID, name, key string) error {
		if c.PluginHost == nil {
			return errNoPluginHost
		}
		return c.PluginHost.SetPluginSecret(ctx, pluginID, name, key)
	}
}

// GetPluginSecretDeleter removes, as the window asks, the key of a secret a
// plugin declares in its manifest, and hands the plugin the secrets it holds now.
func (c Config) GetPluginSecretDeleter() func(ctx context.Context, pluginID, name string) error {
	return func(ctx context.Context, pluginID, name string) error {
		if c.PluginHost == nil {
			return errNoPluginHost
		}
		return c.PluginHost.DeletePluginSecret(ctx, pluginID, name)
	}
}
