package connectionport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Repository is the ordered list of connections kept in the settings file.
type Repository interface {
	// ListConnections returns the valid connections in list order.
	ListConnections() ([]connection.Connection, error)
	// ChangeConnections reads the list, applies change and writes the result as one step. Nothing is
	// written when change fails or leaves the list as it was.
	ChangeConnections(change func([]connection.Connection) ([]connection.Connection, error)) error
}

// Types is the registry of the kinds of connection the installed plugins provide.
type Types interface {
	// ListTypes returns the types of the enabled plugins, in the order of the plugin ids and then of the manifest.
	ListTypes() []connection.Type
	// GetType returns the type with the id, and false where no installed plugin declares it.
	GetType(typeID string) (connection.Type, bool)
}

// OperationTitles names the operations of the connection types in words a person reads.
type OperationTitles interface {
	// ListOperationTitles returns one title for each operation an enabled plugin titles: the first
	// declaration of the name in the order of the plugin ids.
	ListOperationTitles() []manifest.Operation
}

// Uses connects a connection to the settings of rows that name it, and lists the plugins whose grants reach it.
type Uses interface {
	// ListUses returns the settings of rows in force that name the connection, one for each plugin and
	// operation, in the order of the plugin ids.
	ListUses(connectionID string) ([]connection.Use, error)
	// RemoveRows writes the settings of rows without the rows that name the connection.
	RemoveRows(connectionID string) error
	// Refresh sends the configuration again to each running plugin whose rows name the connection.
	Refresh(ctx context.Context, connectionID string) error
	// ListPlugins returns the plugins in force, in the order of the plugin ids.
	ListPlugins() []plugin.Info
}

// ListRows returns the rows a plugin calls an operation through, in the order of use.
type ListRows func(pluginID, operation string) ([]connection.Row, error)

// RowSettings reads the settings of rows of the plugins.
type RowSettings interface {
	// ReadRowSetting returns the setting of rows of the plugin for an operation, with the rows that name an
	// existing connection. A plugin with no such setting shown has no row and does not discover, whatever
	// it is granted.
	ReadRowSetting(pluginID, operation string) (connection.RowSetting, error)
}

// ReportSetAside tells the person that a connection was set aside for a reason and a row after it answered.
type ReportSetAside func(setAside connection.SetAside)

// ResolveConnection reads a connection and its type as a call to the provider needs them. It fails with
// connection.ErrNotFound and connection.ErrTypeUnavailable.
type ResolveConnection func(connectionID string) (connection.Resolved, error)

// ReadConnectionSecrets reads the secret fields of a resolved connection from the key store. It fails with
// connection.ErrSecretMissing for a required secret with no value and connection.ErrStaleRevision for a
// connection that changed since it was resolved.
type ReadConnectionSecrets func(ctx context.Context, resolved connection.Resolved) (map[string]string, error)

// Detected is the connections the providers report ready on this machine through connection.OperationDetected.
type Detected interface {
	// ListDetected returns the connections the providers last reported for the available types, in the order
	// of the types. It does not wait for a provider.
	ListDetected() []connection.Connection
	// IsPending reports whether the type is available, serves detection and its provider has not answered yet
	// in the version of its plugin installed now; an answer kept from an earlier start counts.
	IsPending(typeID string) bool
	// Detect asks the provider of each available type of the plugin that serves detection and waits for the
	// answers.
	Detect(ctx context.Context, pluginID string)
	// Refresh asks every provider not asked lately again in the background.
	Refresh()
}
