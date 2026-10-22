// Package container is the composition root: the only place that knows which
// concrete adapter satisfies which port.
//
// Use cases above it work with the interfaces in core/port. That is the point
// of the arrangement: to replace the index or the vault reader, this package
// changes and nothing else does.
package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/download"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/trash"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"

	"github.com/jiva-studio/numen/modules/libs/core/port/machineport"
	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/modules/libs/core/port/settingsport"
	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/modules/libs/core/port/windowport"
)

// Config is what the user may point somewhere else. Empty fields mean the
// platform's own locations.
type Config struct {
	// IsMockML uses fast synthetic stubs for automated test runs.
	IsMockML     bool
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

	// BookExtensions defines file extensions treated as books. When empty,
	// defaults to all supported document formats.
	BookExtensions []string

	// Performance is the active workload and compute calibration profile.
	Performance settings.Performance

	// ResolvedCompute is the concrete calculated concurrency and batch limits.
	ResolvedCompute settings.ResolvedCompute

	// Hardware specifies hardware specifications detected for the host system.
	Hardware settings.HardwareSpecs

	// Importing configures external URL and media download options.
	Importing download.Config

	// ShouldTranscribe enables automatic transcription for vault recordings without transcripts.
	ShouldTranscribe bool

	// TranscribesUnder defines the maximum byte size for automatic transcription. Zero means no limit.
	TranscribesUnder int64

	// Embedding configures the model used to generate vector embeddings.
	Embedding EmbeddingConfig

	// Proofreading configures the recognition proofreading pipeline.
	Proofreading ProofreadingConfig

	// ScanProofreading and TranscriptProofreading configure automatic proofreading profiles.
	ScanProofreading       Proofread
	TranscriptProofreading Proofread

	// PluginHost coordinates process lifecycle and RPC routing for modular extensions.
	PluginHost pluginport.PluginManager

	// KeyStore provides credential storage. Defaults to the system keychain if nil.
	KeyStore pluginport.KeyStore

	// Texts extracts text from documents and books. Defaults to PluginHost plugins if nil.
	Texts sourceport.TextExtractor

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
}

// NewPluginHost creates and initializes a plugin host for the specified target and search directories.
// The host provisions the dependencies the plugins declare; the ONNX Runtime runs on the
// provider NUMEN_ONNX_PROVIDER names (auto when unset).
func NewPluginHost(target string, dirs ...string) *plugin.Host {
	host := plugin.NewHost(target, dirs...)
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

// ReadSettingsFile returns a function that reads raw settings file contents and path.
func (c Config) ReadSettingsFile() func() (string, string, error) {
	return func() (string, string, error) {
		path, err := c.settingsFile()
		if err != nil {
			return "", "", err
		}
		raw, err := settings.Read(path)
		if err != nil {
			return "", path, err
		}
		return string(raw), path, nil
	}
}

// WriteConfiguredFile replaces the settings file content.
// Returns port.ErrStale if baseline does not match current file content.
func (c Config) WriteConfiguredFile() func(content string, baseline *string) error {
	return func(content string, baseline *string) error {
		path, err := c.settingsFile()
		if err != nil {
			return err
		}
		target := DefaultSettings()
		if err := settings.Write(path, []byte(content), baseline, &target); err != nil {
			return err
		}
		if c.PluginHost != nil {
			if target.Plugins != nil {
				c.PluginHost.SetSettings(target.Plugins)
			}
			_ = c.PluginHost.NotifyAllPluginConfigs(context.Background())
		}
		return nil
	}
}

// NestedHeadingLimitBounds returns min and max bounds for nested headings in notes.
func (Config) NestedHeadingLimitBounds() settings.Bounds {
	return settings.NestedHeadingLimitBounds
}

// GetSettingsWriter patches specific settings paths in the configuration file.
func (c Config) GetSettingsWriter() func(entries []settingsport.Setting) error {
	return func(entries []settingsport.Setting) error {
		target := make([]settings.Setting, 0, len(entries))
		for _, one := range entries {
			var value json.RawMessage
			if err := json.Unmarshal([]byte(one.JSON), &value); err != nil {
				return fmt.Errorf("%w: %w", settingsport.ErrNotASetting, err)
			}
			target = append(target, settings.Setting{At: one.Path, Written: value})
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
}

const pluginSettingsPrefix = "plugins"

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
	if file, selected := c.getPathBeside("numen.json"); selected {
		return file, nil
	}
	return settings.Path()
}

// getPathBeside resolves a file path relative to the registry directory.
func (c Config) getPathBeside(name string) (string, bool) {
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
	return filesystem.VaultReaders{Options: c.vaultOptions()}
}

// VaultWriters opens vaults for write operations.
func (c Config) VaultWriters() vaultport.VaultWriters {
	return filesystem.VaultWriters{Options: c.vaultOptions()}
}

// GetImportedFiles opens files imported from outside the vault.
func (c Config) GetImportedFiles() sourceport.ImportedFiles {
	return filesystem.ImportedFiles{}
}

// VaultWatcher monitors vaults for external filesystem changes.
func (c Config) VaultWatcher() vaultport.VaultWatcher {
	return filesystem.Watcher{Options: c.vaultOptions()}
}

// VaultIdentity provides vault identification.
func (c Config) VaultIdentity() vaultport.VaultIdentity {
	return filesystem.VaultIdentity{Options: c.vaultOptions()}
}

// Trash provides trash and recycling operations.
func (c Config) Trash() machineport.Trash { return trash.New() }

// vaultOptions returns filesystem configuration for vaults.
func (c Config) vaultOptions() filesystem.Options {
	return filesystem.Options{
		ServiceDir:     c.ServiceDir,
		BookExtensions: c.BookExtensions,
	}
}

// GetDerivedStores returns storage for generated artifacts (transcripts, OCR, copies).
func (c Config) GetDerivedStores() vaultport.DerivedStores {
	return filesystem.DerivedStores{
		Options: c.vaultOptions(),
		Area:    filesystem.OCRDir,
		Areas: []string{
			filesystem.TranscriptDir, filesystem.ArticleDir, filesystem.CopyDir, filesystem.IndexDir,
		},
	}
}

// IndexPathOrDefault returns the database index path, defaulting to the system cache directory.
func (c Config) IndexPathOrDefault() (string, error) { return c.indexPath() }

func (c Config) indexPath() (string, error) {
	if c.IndexPath != "" {
		if err := os.MkdirAll(filepath.Dir(c.IndexPath), 0o755); err != nil {
			return "", err
		}
		return c.IndexPath, nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "numen")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "index.db"), nil
}
