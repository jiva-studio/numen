package connectionport

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// MaxNameLength is the longest name of a connection, in characters.
const MaxNameLength = 64

// connectionIDLength is the length of the id of a connection.
const connectionIDLength = 9

// Connection is a stored place to send requests to and the way in. Fields holds the values of the
// fields the person set, secret fields excluded: their values are kept in the key store. Revision
// grows by one at every change, including a change of a secret.
type Connection struct {
	ID                      string
	TypeID                  string
	Name                    string
	Revision                int
	Fields                  map[string]any
	IsPrivateNetworkAllowed bool
}

// Repository is the ordered list of connections kept in the settings file.
type Repository interface {
	// ListConnections returns the valid connections in list order.
	ListConnections() ([]Connection, error)
	// ChangeConnections reads the list, applies change and writes the result as one step. Nothing is
	// written when change fails or leaves the list as it was.
	ChangeConnections(change func([]Connection) ([]Connection, error)) error
}

// The kind of connection a plugin declares, and one field of it, as the plugin catalog library reads
// and validates them.
type (
	ConnectionType  = manifest.ConnectionType
	ConnectionField = manifest.ConnectionField
)

// Type is a kind of connection together with the plugin that provides it. IsAvailable is whether
// that plugin is installed and enabled.
type Type struct {
	PluginID    string
	PluginName  string
	IsAvailable bool
	Declaration ConnectionType
}

// GetID returns the id of the type: the plugin id, a slash and the id the plugin gives the type.
func (t Type) GetID() string {
	return manifest.GetConnectionTypeID(t.PluginID, t.Declaration.ID)
}

// Types is the registry of the kinds of connection the installed plugins provide.
type Types interface {
	// ListTypes returns every declared type, in the order of the plugin ids and then of the manifest.
	ListTypes() []Type
	// GetType returns the type with the id, and false where no installed plugin declares it.
	GetType(typeID string) (Type, bool)
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

// Uses connects a connection to the settings of rows that name it.
type Uses interface {
	// ListUses returns the settings of rows in force that name the connection, one for each plugin and
	// operation, in the order of the plugin ids.
	ListUses(connectionID string) ([]Use, error)
	// RemoveRows writes the settings of rows without the rows that name the connection.
	RemoveRows(connectionID string) error
	// Refresh sends the configuration again to each running plugin whose rows name the connection.
	Refresh(ctx context.Context, connectionID string) error
}
