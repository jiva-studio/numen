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
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"

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

	"github.com/jiva-studio/numen/libs/core/internal/adapter/clock"
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
	IndexPath       string
	RegistryPath    string
	ServiceDir      string
	SettingsPath    string
	ThemesPath      string
	PluginCachePath string

	InterfaceScale, TextScale float64

	ResolvedCompute hardware.ResolvedCompute

	Hardware hardware.Specs

	Chunks *ChunkTokens

	PluginHost pluginport.PluginManager

	PluginCatalog catalog.Config

	PluginDirs []string

	KeyStore pluginport.KeyStore

	TextVersions *textversion.Versions

	TextReaders textport.TextReaders

	ShouldRebuildIndex bool

	SafeMode bool

	ErrorHandler windowport.ErrorHandler

	Now machineport.Clock

	Resources *Resources

	agentProvider agentport.AgentProvider
	programs      *settings.ProgramList
}

// NewPluginHost creates and initializes a plugin host for the specified target and search directories.
// The host provisions the dependencies the plugins declare; the ONNX Runtime runs on the
// provider NUMEN_ONNX_PROVIDER names (auto when unset).
func NewPluginHost(target string, dirs ...string) *plugin.Host {
	host := plugin.NewHost(clock.System{}, target, dirs...)
	host.SetContentTypeTable(sharedmimeinfo.Open())
	host.SetDependencies(newDependencies(""))
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
	return clock.System{}
}

func (c Config) handleError(err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
}

// SyncSetting returns whether note title and filename synchronization is enabled.
func (c Config) SyncSetting() note.SyncSetting {
	return func() note.SyncTitleAndFilename {
		target, err := c.Settings()
		if err != nil {
			c.handleError(fmt.Errorf("read settings: %w", err))
		}
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
		target, err := c.getCheckedSettingsAt(path)
		if err != nil {
			return "", path, err
		}
		content, err := target.WriteJSON()
		return content, path, err
	}
}

// GetSettingsWriter patches specific settings paths in the configuration file. The readers of the
// setting-changed event receive the settings whose value changed.
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
	if err := c.checkWrites(entries); err != nil {
		return fmt.Errorf("%w: %w", settingsport.ErrNotASetting, err)
	}
	path, err := c.settingsFile()
	if err != nil {
		return err
	}
	into := DefaultSettings()
	if err := c.getResources().settingsFiles.Save(path, &into, target...); err != nil {
		return err
	}
	c.refreshSettingEvents()
	return nil
}

func (c Config) checkWrites(entries []settingsport.Setting) error {
	declarations := c.getDeclarations()
	registries := c.getRegistries()
	for _, one := range entries {
		var val any
		if err := json.Unmarshal([]byte(one.JSON), &val); err != nil {
			return err
		}
		if err := settings.CheckWrite(declarations, registries, one.Path, val); err != nil {
			return err
		}
		if c.PluginHost != nil && isPluginSetting(one.Path) {
			if err := c.PluginHost.CheckPluginSetting(one.Path[0], one.Path[1], val); err != nil {
				return err
			}
		}
	}
	return nil
}

func isPluginSetting(path []string) bool {
	return len(path) == 2 && path[0] != settings.OwnerApp
}

func notifyPluginSettings(host pluginport.PluginConfigurator, entries []settingsport.Setting, report func(error)) {
	if host == nil {
		return
	}
	// The enabled entries go last: enabling a plugin generates the values of its settings that hold
	// none, and a value in the same batch is in memory by then.
	isEnabledEntry := func(one settingsport.Setting) bool {
		return len(one.Path) == 2 && one.Path[1] == hostFieldEnabled
	}
	ordered := slices.Concat(
		slices.DeleteFunc(slices.Clone(entries), isEnabledEntry),
		slices.DeleteFunc(slices.Clone(entries), func(one settingsport.Setting) bool { return !isEnabledEntry(one) }),
	)
	for _, one := range ordered {
		if !isPluginSetting(one.Path) {
			continue
		}
		var val any
		if err := json.Unmarshal([]byte(one.JSON), &val); err == nil {
			if err := host.UpdatePluginSetting(context.Background(), one.Path[0], one.Path[1], val); err != nil {
				report(fmt.Errorf("update plugin setting %s: %w", strings.Join(one.Path, "."), err))
			}
		}
	}
}

func (c Config) settingsFile() (string, error) {
	if c.SettingsPath != "" {
		return c.SettingsPath, nil
	}
	if file, selected := c.getAdjacentPath("numen.json"); selected {
		return file, nil
	}
	return settings.Path()
}

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
	return vaultwriter.VaultWriters{Options: c.vaultOptions(), Locks: c.getResources().vaultLocks}
}

// GetImportedFiles opens files imported from outside the vault.
func (c Config) GetImportedFiles() sourceport.ImportedFiles {
	return filesystem.ImportedFiles{}
}

// VaultWatcher monitors vaults for external filesystem changes.
func (c Config) VaultWatcher() vaultport.VaultWatcher {
	return vaultwatch.Watcher{Options: c.vaultOptions(), Hub: c.getResources().watches}
}

// VaultIdentity provides vault identification.
func (c Config) VaultIdentity() vaultport.VaultIdentity {
	return vaultreader.Identity{Options: c.vaultOptions()}
}

// Trash provides trash and recycling operations.
func (c Config) Trash() machineport.Trash { return trash.New(c.Clock()) }

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
