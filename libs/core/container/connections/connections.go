// Package connections wires the use cases on the connections to services and programs.
package connections

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	settingsadapter "github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/connectiontypes"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
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
	RefreshSettingEvents func()
}

// OpenConnections returns the use cases on the connections, or nil for an installation that has no
// plugin host and so no kind of connection.
func (c Config) OpenConnections() *connection.Connections {
	if c.PluginHost == nil {
		return nil
	}
	repository := settingsadapter.NewConnectionRepository(c.SettingsFiles, c.SettingsFile, c.ReadCheckedSettings, c.RefreshSettingEvents)
	return connection.NewConnections(repository, connectiontypes.NewRegistry(c.PluginHost), c.KeyStore, NewConnectionID)
}

// NewConnectionID generates a random id for a connection: "c" and 8 lowercase hexadecimal digits.
func NewConnectionID() (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("read random bytes for the id of a connection: %w", err)
	}
	return "c" + hex.EncodeToString(random[:]), nil
}
