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

// OperationDetected is the operation through which a provider reports the connections of a type it found
// ready on this machine. The core owns this one name; a type takes part by listing it among its operations.
const OperationDetected = "connections.detected"

const detectedIDSeparator = "/"

// GetDetectedID returns the id of a detected connection: the type id, a slash and the name the provider
// reported. No stored connection has a slash in its id.
func GetDetectedID(typeID, name string) string {
	return typeID + detectedIDSeparator + name
}

// IsDetected reports whether the id has the shape GetDetectedID gives: a plugin id, a type and a name joined by
// slashes.
func IsDetected(id string) bool {
	_, _, isDetected := SplitDetectedID(id)
	return isDetected
}

// SplitDetectedID cuts the id at its first two slashes; the name may hold more of them.
func SplitDetectedID(id string) (typeID, name string, isDetected bool) {
	pluginID, rest, _ := strings.Cut(id, detectedIDSeparator)
	localID, name, _ := strings.Cut(rest, detectedIDSeparator)
	if pluginID == "" || localID == "" || name == "" {
		return "", "", false
	}
	return manifest.GetConnectionTypeID(pluginID, localID), name, true
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

// Use is a setting of rows of a plugin that names a connection. Operations are the operations the
// setting is used for.
type Use struct {
	PluginID   string
	PluginName string
	Operations []string
}

// Row is one entry of the settings of a plugin that names a connection, with the values of the row
// fields of the type chosen for it. IsDiscovered is a row the person did not choose: the setting discovers,
// and the row stands for a connection that serves the operation.
type Row struct {
	ConnectionID string
	Options      map[string]string
	IsDiscovered bool
}

// RowSetting is the setting of rows a plugin uses for an operation. Rows are the rows the person chose, in
// their order. IsDiscovering is whether, while Rows is empty, the plugin uses every connection whose type
// serves the operation.
type RowSetting struct {
	Rows          []Row
	IsDiscovering bool
}
