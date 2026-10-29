// Package container is the composition root: the only place that selects which
// concrete adapter satisfies which port.
//
// Use cases above it work with the interfaces in core/port. To replace the index or
// the vault reader, only this package changes.
package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/trash"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// Config is what the user may point somewhere else. Empty fields mean the
// platform's own locations.
type Config struct {
	// IndexPath is the index database. The application chooses it; the core
	// does not start without one.
	IndexPath    string
	RegistryPath string
	ServiceDir   string
	// SettingsPath specifies the configuration file path.
	SettingsPath string
	// ThemesPath specifies the directory for user themes.
	ThemesPath string
	// PluginCachePath specifies the cache directory for plugin vault data.
	PluginCachePath string

	// InterfaceScale and TextScale configure UI and text rendering scales.
	// When non-zero, they override values defined in the settings file.
	InterfaceScale, TextScale float64

	// Performance is the active workload and compute calibration profile.
	Performance hardware.Performance

	// ResolvedCompute is the concrete calculated concurrency and batch limits.
	ResolvedCompute hardware.ResolvedCompute

	// Hardware specifies hardware specifications detected for the host system.
	Hardware hardware.Specs

	// Embedding is how the index stores and queries vectors.
	Embedding EmbeddingConfig

	// PluginHost coordinates process lifecycle and RPC routing for modular extensions.
	PluginHost pluginport.PluginManager

	// KeyStore is where the keys of plugin secrets are kept. The application names it; without one
	// every secret fails with port.ErrNoKeyStore.
	KeyStore pluginport.KeyStore

	// TextVersions is the text versions of the vaults' files, which the index, the
	// search and the highlighting read text through. Nil is none, and only notes
	// have text. NewTextVersions makes it.
	TextVersions *textversion.Versions

	// TextReaders is where the index, the search and the highlighting read text
	// versions. Defaults to TextVersions if nil.
	TextReaders textport.TextReaders

	// Agent configures the active assistant model.
	Agent settings.Agent

	// ShouldRebuildIndex forces re-indexing of all vault files regardless of index state.
	ShouldRebuildIndex bool

	// SafeMode disables loading non-essential plugins for emergency recovery.
	SafeMode bool

	// ErrorHandler receives asynchronous errors.
	ErrorHandler windowport.ErrorHandler

	// Now provides current time for timestamping and scheduling. Defaults to time.Now if nil.
	Now machineport.Clock

	agentProvider agentport.AgentProvider
}

// NewPluginHost creates and initializes a plugin host for the specified target and search directories.
// The host provisions the dependencies the plugins declare; the ONNX Runtime runs on the
// provider NUMEN_ONNX_PROVIDER names (auto when unset).
func NewPluginHost(target string, dirs ...string) *plugin.Host {
	host := plugin.NewHost(target, dirs...)
	host.SetContentTypeTable(sharedmimeinfo.Open())
	host.SetDependencies(newDependencies())
	if err := host.Discover(); err != nil {
		fmt.Fprintf(os.Stderr, "numen: discover plugins: %v\n", err)
	}
	return host
}

// Clock is the clock every scenario assembled here is handed.

func (c Config) Clock() machineport.Clock {
	if c.Now != nil {
		return c.Now
	}
	return time.Now
}

// handleError forwards errors to ErrorHandler if configured.
func (c Config) handleError(err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
}

// SyncSetting returns whether note title and filename synchronization is enabled.
func (c Config) SyncSetting() note.SyncSetting {
	return func() note.SyncTitleAndFilename {
		target, _ := c.Settings()
		return note.SyncTitleAndFilename(target.Sync())
	}
}

// ReadSettings returns a function that reads settings as JSON and returns the settings file path.
func (c Config) ReadSettings() func() (string, string, error) {
	return func() (string, string, error) {
		path, err := c.settingsFile()
		if err != nil {
			return "", "", err
		}
		target, err := c.getSettingsAt(path)
		if err != nil {
			return "", path, err
		}
		content, err := target.WriteJSON()
		return content, path, err
	}
}

// GetSettingsWriter patches specific settings paths in the configuration file.
func (c Config) GetSettingsWriter() func(entries []settingsport.Setting) error {
	return c.writeSettings
}

func (c Config) writeSettings(entries []settingsport.Setting) error {
	target := make([]settings.Setting, 0, len(entries))
	for _, one := range entries {
		var value json.RawMessage
		if err := json.Unmarshal([]byte(one.JSON), &value); err != nil {
			return fmt.Errorf("%w: %w", settingsport.ErrNotASetting, err)
		}
		target = append(target, settings.Setting{At: one.Path, Written: value})
	}
	if err := checkPluginSettings(c.PluginHost, entries); err != nil {
		return fmt.Errorf("%w: %w", settingsport.ErrNotASetting, err)
	}
	path, err := c.settingsFile()
	if err != nil {
		return err
	}
	into := DefaultSettings()
	if err := settings.Save(path, &into, target...); err != nil {
		return err
	}
	notifyPluginSettings(c.PluginHost, entries)
	return nil
}

const pluginSettingsPrefix = "plugins"

// checkPluginSettings reports the first plugin setting among the entries that does not take its value.
func checkPluginSettings(host pluginport.PluginConfigurator, entries []settingsport.Setting) error {
	if host == nil {
		return nil
	}
	for _, one := range entries {
		if len(one.Path) != 3 || one.Path[0] != pluginSettingsPrefix {
			continue
		}
		var val any
		if err := json.Unmarshal([]byte(one.JSON), &val); err != nil {
			return err
		}
		if err := host.CheckPluginSetting(one.Path[1], one.Path[2], val); err != nil {
			return err
		}
	}
	return nil
}

func notifyPluginSettings(host pluginport.PluginConfigurator, entries []settingsport.Setting) {
	if host == nil {
		return
	}
	for _, one := range entries {
		if len(one.Path) < 3 || one.Path[0] != pluginSettingsPrefix {
			continue
		}
		var val any
		if err := json.Unmarshal([]byte(one.JSON), &val); err == nil {
			_ = host.UpdatePluginSetting(context.Background(), one.Path[1], one.Path[2], val)
		}
	}
}

// settingsFile resolves the configuration file path.
func (c Config) settingsFile() (string, error) {
	if c.SettingsPath != "" {
		return c.SettingsPath, nil
	}
	if file, selected := c.getAdjacentPath("numen.json"); selected {
		return file, nil
	}
	return settings.Path()
}

// getAdjacentPath resolves a file path relative to the registry directory.
func (c Config) getAdjacentPath(name string) (string, bool) {
	if c.RegistryPath == "" {
		return "", false
	}
	return filepath.Join(filepath.Dir(c.RegistryPath), name), true
}

// Registry opens the vault registry for this installation.
func (c Config) Registry() (vaultport.VaultRegistry, error) {
	if c.RegistryPath != "" {
		return appstate.OpenAt(c.RegistryPath), nil
	}
	return appstate.Open()
}

// VaultReaders opens vaults for reading.
func (c Config) VaultReaders() vaultport.VaultReaders {
	return vaultreader.Readers{Options: c.vaultOptions()}
}

// VaultWriters opens vaults for write operations.
func (c Config) VaultWriters() vaultport.VaultWriters {
	return vaultwriter.VaultWriters{Options: c.vaultOptions()}
}

// GetImportedFiles opens files imported from outside the vault.
func (c Config) GetImportedFiles() sourceport.ImportedFiles {
	return filesystem.ImportedFiles{}
}

// VaultWatcher monitors vaults for external filesystem changes.
func (c Config) VaultWatcher() vaultport.VaultWatcher {
	return vaultwatch.Watcher{Options: c.vaultOptions()}
}

// VaultIdentity provides vault identification.
func (c Config) VaultIdentity() vaultport.VaultIdentity {
	return vaultreader.Identity{Options: c.vaultOptions()}
}

// Trash provides trash and recycling operations.
func (c Config) Trash() machineport.Trash { return trash.New() }

// vaultOptions returns filesystem configuration for vaults.
func (c Config) vaultOptions() vaultconfig.Options {
	return vaultconfig.Options{
		ServiceDir: c.ServiceDir,
	}
}

// ErrNoIndexPath is a configuration that names no index database.
var ErrNoIndexPath = errors.New("the configuration names no index database: the application sets IndexPath")

func (c Config) indexPath() (string, error) {
	if c.IndexPath == "" {
		return "", ErrNoIndexPath
	}
	if err := os.MkdirAll(filepath.Dir(c.IndexPath), 0o755); err != nil {
		return "", err
	}
	return c.IndexPath, nil
}

// GetWait is the wait every scenario assembled here is handed, bound to this
// machine's timers.
func (c Config) GetWait() machineport.Wait {
	return func(ctx context.Context, d time.Duration) error {
		if d <= 0 {
			return ctx.Err()
		}
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
}

// GetDelay is the delayed call every scenario assembled here is handed, bound
// to this machine's timers.
func (c Config) GetDelay() machineport.Delay {
	return func(d time.Duration, f func()) func() bool {
		return time.AfterFunc(d, f).Stop
	}
}
