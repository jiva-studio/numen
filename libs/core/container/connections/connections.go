// Package connections wires the use cases on the connections to services and programs.
package connections

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	settingsadapter "github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/connectiontypes"
	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/usecase/connection"
)

// Config wires the connections of an installation: the list in the settings file, the kinds of
// connection the plugin host lists and the key store.
type Config struct {
	PluginHost           pluginport.PluginManager
	KeyStore             pluginport.KeyStore
	SettingsFiles        *settingsadapter.Files
	SettingsFile         func() (string, error)
	ReadCheckedSettings  func() (settingsadapter.File, error)
	ReadStoredSettings   func() (settingsadapter.File, error)
	RefreshSettingEvents func()
	Plugins              func() []pluginport.PluginInfo
	GetDeclarations      settingsadapter.Declarations
	WriteSettings        func(entries []settingsport.Setting) error
	HandleError          func(err error)
}

func (c Config) openRepository() connectionport.Repository {
	return settingsadapter.NewConnectionRepository(c.SettingsFiles, c.SettingsFile, c.ReadCheckedSettings, c.RefreshSettingEvents)
}

// OpenRows returns the rows of the settings of the plugins, which are the connections each plugin
// may use, and the link between a connection and the settings that name it. It is nil for an
// installation that has no plugin host.
func (c Config) OpenRows() interface {
	connectionport.Rows
	connectionport.Uses
} {
	if c.PluginHost == nil {
		return nil
	}
	return settingsadapter.NewConnectionRows(settingsadapter.ConnectionRowsSource{
		Connections: c.openRepository(),
		Plugins:     c.Plugins,
		Declared:    c.GetDeclarations,
		ReadStored:  c.ReadStoredSettings,
		Write:       c.WriteSettings,
		Refresh:     c.PluginHost.Refresh,
		Report:      c.HandleError,
	})
}

// OpenConnections returns the use cases on the connections, or nil for an installation that has no
// plugin host and so no kind of connection.
func (c Config) OpenConnections() *connection.Connections {
	if c.PluginHost == nil {
		return nil
	}
	return connection.NewConnections(c.openRepository(), connectiontypes.NewRegistry(c.PluginHost), c.KeyStore, c.OpenRows(), NewConnectionID)
}

// NewConnectionID generates a random id for a connection: "c" and 8 lowercase hexadecimal digits.
func NewConnectionID() (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("read random bytes for the id of a connection: %w", err)
	}
	return "c" + hex.EncodeToString(random[:]), nil
}
