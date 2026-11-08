package plugins

import (
	"context"
	"slices"

	settingsadapter "github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/assembly"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/localsource"
)

// Config wires the plugin catalog, plugin host, agent provider and dependencies.
type Config struct {
	PluginCachePath       string
	ServiceDir            string
	SafeMode              bool
	PluginHost            pluginport.PluginManager
	PluginCatalog         catalog.Config
	PluginDirs            []string
	KeyStore              pluginport.KeyStore
	SettingsFiles         *settingsadapter.Files
	SecretGate            *secret.Gate
	Programs              *settingsadapter.ProgramList
	VaultReaders          vaultport.VaultReaders
	VaultWriters          vaultport.VaultWriters
	VaultWatcher          vaultport.VaultWatcher
	ErrorHandler          func(error)
	SettingsFile          func() (string, error)
	ReadCheckedSettings   func() (settingsadapter.File, error)
	GetSettingsOrDefaults func() settingsadapter.File
	RefreshSettingEvents  func()
	MakeRefresh           func(*vault.Index, noteport.NoteRepository) vault.Refresh
	BuildNotes            func(noteport.NoteQueries, noteport.LinkQueries, sourceport.SourceRepository, sourceport.SourceQueries, note.IndexUpdater) note.Scenarios
	Moves                 func(context.Context) (<-chan struct{}, error)
}

// GetPluginCatalog is the catalog of the plugins on disk, the disabled ones too, and of the sources
// the configuration names. It is nil for an
// installation that names no target or has no plugin host.
func (c Config) GetPluginCatalog() *catalog.Service {
	if c.PluginCatalog.Target == "" || c.PluginHost == nil {
		return nil
	}
	config := c.PluginCatalog
	config.Sources = append(
		[]catalog.Source{localsource.New(slices.Clone(c.PluginDirs), localsource.SettingsFunc(c.PluginHost.IsPluginEnabled))},
		config.Sources...)
	service, err := catalog.NewService(config)
	if err != nil {
		c.handleError(err)
		return nil
	}
	return service
}

// GetEditorCatalog returns the plugin catalog formatted for the editor assembly.
func (c Config) GetEditorCatalog() assembly.PluginCatalog {
	service := c.GetPluginCatalog()
	if service == nil {
		return nil
	}
	return service
}

func (c Config) handleError(err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
}
