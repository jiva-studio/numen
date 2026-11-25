package connection

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

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

// RowSettings reads the settings of rows of the plugins.
type RowSettings interface {
	// ReadRowSetting returns the setting of rows of the plugin for an operation, with the rows that name an
	// existing connection. A plugin with no such setting shown has no row and does not discover, whatever
	// it is granted.
	ReadRowSetting(pluginID, operation string) (connection.RowSetting, error)
}

// OperationTitles names the operations of the connection types in words a person reads.
type OperationTitles interface {
	// ListOperationTitles returns one title for each operation an enabled plugin titles: the first
	// declaration of the name in the order of the plugin ids.
	ListOperationTitles() []manifest.Operation
}

// ReadConnectionSecrets reads the secret fields of a resolved connection from the key store. It fails with
// connection.ErrSecretMissing for a required secret with no value and connection.ErrStaleRevision for a
// connection that changed since it was resolved.
type ReadConnectionSecrets func(ctx context.Context, resolved connection.Resolution) (map[string]string, error)
