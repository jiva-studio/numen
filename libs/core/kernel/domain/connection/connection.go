// Package connection holds what a stored connection is: its identity, its kind, and the rows of plugin
// settings that name it.
package connection

import (
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// MaxNameLength is the longest name of a connection, in characters.
const MaxNameLength = 64

const connectionIDLength = 9

// Connection is a stored place to send requests to and the way in. Fields holds the values of the
// fields the person set, secret fields excluded: their values are kept in the key store. Revision
// grows by one at every change, including a change of a secret.
type Connection struct {
	ID       string
	TypeID   string
	Name     string
	Revision int
	Fields   map[string]any
}

// Type is a kind of connection together with the plugin that provides it. IsAvailable is whether
// that plugin is installed and enabled.
type Type struct {
	PluginID    string
	PluginName  string
	IsAvailable bool
	Declaration manifest.ConnectionType
}

// GetID returns the id of the type: the plugin id, a slash and the id the plugin gives the type.
func (t Type) GetID() string {
	return manifest.GetConnectionTypeID(t.PluginID, t.Declaration.ID)
}

// IsValidName reports whether the text is a name of a connection: not empty, without leading or
// trailing space, and at most MaxNameLength characters.
func IsValidName(name string) bool {
	return name != "" && strings.TrimSpace(name) == name && utf8.RuneCountInString(name) <= MaxNameLength
}

// IsValidID reports whether the text has the shape of the id of a connection: "c" and 8 lowercase
// hexadecimal digits.
func IsValidID(id string) bool {
	if len(id) != connectionIDLength || id[0] != 'c' {
		return false
	}
	for _, digit := range id[1:] {
		if !strings.ContainsRune("0123456789abcdef", digit) {
			return false
		}
	}
	return true
}

// Use is a setting of rows of a plugin that names a connection. Operation is the operation the
// setting needs.
type Use struct{ PluginID, PluginName, Operation string }

// Row is one entry of the settings of a plugin that names a connection, with the model and the
// reasoning level chosen for it.
type Row struct{ ConnectionID, Model, Reasoning string }
