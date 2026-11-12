package connectionport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
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
	// ListTypes returns every declared type, in the order of the plugin ids and then of the manifest.
	ListTypes() []connection.Type
	// GetType returns the type with the id, and false where no installed plugin declares it.
	GetType(typeID string) (connection.Type, bool)
}

// Uses connects a connection to the settings of rows that name it.
type Uses interface {
	// ListUses returns the settings of rows in force that name the connection, one for each plugin and
	// operation, in the order of the plugin ids.
	ListUses(connectionID string) ([]connection.Use, error)
	// RemoveRows writes the settings of rows without the rows that name the connection.
	RemoveRows(connectionID string) error
	// Refresh sends the configuration again to each running plugin whose rows name the connection.
	Refresh(ctx context.Context, connectionID string) error
}

// Rows reads the rows of a plugin in the order of use.
type Rows interface {
	// ListRows returns the rows of the plugin for an operation, in the order the person set.
	ListRows(pluginID, operation string) ([]connection.Row, error)
}

// Notices tells the person what the core did on its own.
type Notices interface {
	// ReportSetAside says that a connection was set aside for a reason and a row after it answered.
	ReportSetAside(connectionID, connectionName string, reason connection.FailureReason)
}

// ResolveConnection reads a connection and its type as a call to the provider needs them. It fails with
// connection.ErrNotFound and connection.ErrTypeUnavailable.
type ResolveConnection func(connectionID string) (connection.Resolved, error)

// ReadConnectionSecrets reads the secret fields of a resolved connection from the key store. It fails with
// connection.ErrSecretMissing for a required secret with no value and connection.ErrStaleRevision for a
// connection that changed since it was resolved.
type ReadConnectionSecrets func(ctx context.Context, resolved connection.Resolved) (map[string]string, error)
